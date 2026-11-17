package com.dhaba.kitchen.config;

import java.time.Duration;
import org.springframework.boot.context.properties.ConfigurationProperties;

/** Worker settings bound from the {@code worker.*} configuration namespace. */
@ConfigurationProperties(prefix = "worker")
public record WorkerProperties(
        String id,
        String version,
        int maxAttempts,
        Duration baseBackoff,
        Duration maxBackoff,
        Duration pollTimeout,
        Duration heartbeatInterval,
        Duration heartbeatTtl,
        String heartbeatFile,
        boolean autoAdvance,
        Duration acceptDelay,
        Duration cookTime) {

    public WorkerProperties {
        if (id == null || id.isBlank()) {
            throw new IllegalArgumentException("worker.id must not be blank");
        }
        if (maxAttempts < 1) {
            throw new IllegalArgumentException("worker.max-attempts must be >= 1");
        }
    }

    /** Exponential backoff for the given (zero-based) attempt, capped at maxBackoff. */
    public Duration backoffFor(int attempt) {
        long base = baseBackoff.toMillis();
        int shift = Math.min(Math.max(attempt, 0), 20);
        long millis = Math.min(base * (1L << shift), maxBackoff.toMillis());
        return Duration.ofMillis(millis);
    }
}
