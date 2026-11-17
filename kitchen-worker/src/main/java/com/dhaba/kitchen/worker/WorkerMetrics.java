package com.dhaba.kitchen.worker;

import java.time.Instant;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;
import org.springframework.stereotype.Component;

/** In-process counters reported in the heartbeat. */
@Component
public class WorkerMetrics {

    private final AtomicLong processed = new AtomicLong();
    private final AtomicLong retried = new AtomicLong();
    private final AtomicLong deadLettered = new AtomicLong();
    private final AtomicReference<Instant> lastJobAt = new AtomicReference<>();

    void processed() {
        processed.incrementAndGet();
        lastJobAt.set(Instant.now());
    }

    void retried() {
        retried.incrementAndGet();
    }

    void deadLettered() {
        deadLettered.incrementAndGet();
    }

    public long processedCount() {
        return processed.get();
    }

    public long retriedCount() {
        return retried.get();
    }

    public long deadLetteredCount() {
        return deadLettered.get();
    }

    public Instant lastJobAt() {
        return lastJobAt.get();
    }
}
