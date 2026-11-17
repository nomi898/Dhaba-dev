package com.dhaba.kitchen.worker;

import com.dhaba.kitchen.config.WorkerProperties;
import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.PermanentJobException;
import com.dhaba.kitchen.queue.JobCodec;
import com.dhaba.kitchen.queue.JobQueue;
import java.time.Duration;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.SmartLifecycle;
import org.springframework.stereotype.Component;

/**
 * Consumer loop. Implements {@link SmartLifecycle} so that on SIGTERM it stops taking new jobs,
 * finishes the job in hand, and only then lets the application context close.
 */
@Component
public class QueueConsumer implements SmartLifecycle {

    private static final Logger LOG = LoggerFactory.getLogger(QueueConsumer.class);
    private static final Duration ERROR_PAUSE = Duration.ofSeconds(1);

    private final JobQueue queue;
    private final JobCodec codec;
    private final JobProcessor processor;
    private final WorkerProperties props;
    private final WorkerMetrics metrics;

    private volatile boolean running;
    private volatile CountDownLatch stopped = new CountDownLatch(0);
    private Thread thread;

    public QueueConsumer(JobQueue queue, JobCodec codec, JobProcessor processor, WorkerProperties props,
            WorkerMetrics metrics) {
        this.queue = queue;
        this.codec = codec;
        this.processor = processor;
        this.props = props;
        this.metrics = metrics;
    }

    @Override
    public synchronized void start() {
        if (running) {
            return;
        }
        running = true;
        stopped = new CountDownLatch(1);
        thread = Thread.ofPlatform().name("queue-consumer").start(this::loop);
        LOG.info("kitchen-worker {} consuming jobs (autoAdvance={}, maxAttempts={})", props.id(),
                props.autoAdvance(), props.maxAttempts());
    }

    @Override
    public void stop(Runnable callback) {
        running = false;
        LOG.info("shutdown requested; finishing in-flight job");
        Thread.ofVirtual().start(() -> {
            try {
                if (!stopped.await(props.pollTimeout().toMillis() + 25_000, TimeUnit.MILLISECONDS)) {
                    LOG.warn("consumer did not stop in time; in-flight job will be recovered by a peer");
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            } finally {
                LOG.info("consumer stopped");
                callback.run();
            }
        });
    }

    @Override
    public void stop() {
        stop(() -> { });
    }

    @Override
    public boolean isRunning() {
        return running;
    }

    private void loop() {
        try {
            queue.recoverOwn();
        } catch (RuntimeException e) {
            LOG.warn("could not recover own processing list: {}", e.getMessage());
        }
        try {
            while (running) {
                try {
                    Optional<String> raw = queue.take(props.pollTimeout());
                    raw.ifPresent(this::handle);
                } catch (RuntimeException e) {
                    LOG.error("queue error: {}", e.getMessage());
                    pause();
                }
            }
        } finally {
            stopped.countDown();
        }
    }

    /** Process one raw job: ack on success, retry with backoff on transient failure, dead-letter otherwise. */
    void handle(String raw) {
        Job job;
        try {
            job = codec.decode(raw);
        } catch (PermanentJobException e) {
            LOG.error("dead-lettering malformed job: {}", e.getMessage());
            queue.deadLetter(raw, e.getMessage());
            metrics.deadLettered();
            return;
        }
        try {
            processor.process(job);
            queue.ack(raw);
            metrics.processed();
        } catch (PermanentJobException e) {
            LOG.error("job {} for order {} failed permanently: {}", job.id(), job.orderId(), e.getMessage());
            queue.deadLetter(raw, e.getMessage());
            metrics.deadLettered();
        } catch (RuntimeException e) {
            int attemptsMade = job.attempt() + 1;
            if (attemptsMade >= props.maxAttempts()) {
                LOG.error("job {} for order {} failed after {} attempts: {}", job.id(), job.orderId(),
                        attemptsMade, e.getMessage());
                queue.deadLetter(raw, "max attempts exceeded: " + e.getMessage());
                metrics.deadLettered();
                return;
            }
            Duration delay = props.backoffFor(job.attempt());
            LOG.warn("job {} for order {} failed (attempt {}/{}), retrying in {}ms: {}", job.id(), job.orderId(),
                    attemptsMade, props.maxAttempts(), delay.toMillis(), e.getMessage());
            queue.retry(raw, job.nextAttempt(), delay);
            metrics.retried();
        }
    }

    private void pause() {
        try {
            Thread.sleep(ERROR_PAUSE);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            running = false;
        }
    }
}
