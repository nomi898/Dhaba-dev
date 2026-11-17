package com.dhaba.kitchen.worker;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.dhaba.kitchen.TestFixtures;
import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.JobType;
import com.dhaba.kitchen.model.PermanentJobException;
import com.dhaba.kitchen.queue.JobCodec;
import com.dhaba.kitchen.queue.JobQueue;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.dao.QueryTimeoutException;

class QueueConsumerTest {

    private JobQueue queue;
    private JobCodec codec;
    private JobProcessor processor;
    private WorkerMetrics metrics;
    private QueueConsumer consumer;

    @BeforeEach
    void setUp() {
        queue = mock(JobQueue.class);
        codec = mock(JobCodec.class);
        processor = mock(JobProcessor.class);
        metrics = new WorkerMetrics();
        consumer = new QueueConsumer(queue, codec, processor, TestFixtures.props(false), metrics);
    }

    private Job job(int attempt) {
        return new Job("j", JobType.ORDER_CREATED, 1, null, "orders-api", attempt, Instant.now());
    }

    @Test
    void successfulJobIsAcked() {
        when(codec.decode("raw")).thenReturn(job(0));
        consumer.handle("raw");
        verify(queue).ack("raw");
        assertThat(metrics.processedCount()).isEqualTo(1);
    }

    @Test
    void transientFailureIsRetriedWithBackoff() {
        when(codec.decode("raw")).thenReturn(job(1));
        doThrow(new QueryTimeoutException("db slow")).when(processor).process(any());
        consumer.handle("raw");
        verify(queue).retry(eq("raw"), argThat(j -> j.attempt() == 2), eq(Duration.ofSeconds(2)));
        verify(queue, never()).ack(anyString());
        assertThat(metrics.retriedCount()).isEqualTo(1);
    }

    @Test
    void exhaustedRetriesAreDeadLettered() {
        when(codec.decode("raw")).thenReturn(job(2));
        doThrow(new IllegalStateException("boom")).when(processor).process(any());
        consumer.handle("raw");
        verify(queue).deadLetter(eq("raw"), argThat(r -> r.contains("max attempts")));
        verify(queue, never()).retry(anyString(), any(), any());
    }

    @Test
    void permanentFailureIsDeadLetteredImmediately() {
        when(codec.decode("raw")).thenReturn(job(0));
        doThrow(new PermanentJobException("order gone")).when(processor).process(any());
        consumer.handle("raw");
        verify(queue).deadLetter("raw", "order gone");
        assertThat(metrics.deadLetteredCount()).isEqualTo(1);
    }

    @Test
    void malformedJobIsDeadLettered() {
        when(codec.decode("junk")).thenThrow(new PermanentJobException("unparseable"));
        consumer.handle("junk");
        verify(queue).deadLetter("junk", "unparseable");
        verify(processor, never()).process(any());
    }

    @Test
    void gracefulStopFinishesAndInvokesCallback() throws InterruptedException {
        when(queue.take(any())).thenReturn(Optional.empty());
        consumer.start();
        assertThat(consumer.isRunning()).isTrue();
        CountDownLatch done = new CountDownLatch(1);
        consumer.stop(done::countDown);
        assertThat(done.await(5, TimeUnit.SECONDS)).isTrue();
        assertThat(consumer.isRunning()).isFalse();
        verify(queue).recoverOwn();
    }
}
