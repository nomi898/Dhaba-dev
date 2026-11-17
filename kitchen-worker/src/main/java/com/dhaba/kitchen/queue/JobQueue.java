package com.dhaba.kitchen.queue;

import com.dhaba.kitchen.model.Job;
import java.time.Duration;
import java.util.Optional;

/** Reliable job queue operations. */
public interface JobQueue {

    /** Block up to {@code timeout} for a job; the raw payload is held in this worker's processing list. */
    Optional<String> take(Duration timeout);

    /** Job finished: drop it from the processing list. */
    void ack(String raw);

    /** Job failed transiently: schedule {@code next} after {@code delay} and drop {@code raw}. */
    void retry(String raw, Job next, Duration delay);

    /** Job can never succeed: move it to the dead-letter list. */
    void deadLetter(String raw, String reason);

    /** Schedule a new job to run after {@code delay}. */
    void schedule(Job job, Duration delay);

    /** Move due delayed jobs onto the main queue. Returns how many were moved. */
    long promoteDue();

    /** Return jobs held by dead workers (expired heartbeat) to the queue. */
    long recoverOrphans();

    /** On startup, return any jobs left in this worker's own processing list. */
    long recoverOwn();
}
