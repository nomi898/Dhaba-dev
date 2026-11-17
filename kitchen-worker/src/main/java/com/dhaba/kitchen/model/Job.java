package com.dhaba.kitchen.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import java.time.Instant;
import java.util.UUID;

/** Queue message. Field names match the JSON contract written by orders-api. */
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonInclude(JsonInclude.Include.NON_NULL)
public record Job(
        String id,
        JobType type,
        long orderId,
        OrderStatus targetStatus,
        String source,
        int attempt,
        Instant createdAt) {

    public static Job statusChange(long orderId, OrderStatus target, String source) {
        return new Job(UUID.randomUUID().toString(), JobType.STATUS_CHANGE, orderId, target, source, 0, Instant.now());
    }

    public Job nextAttempt() {
        return new Job(id, type, orderId, targetStatus, source, attempt + 1, createdAt);
    }
}
