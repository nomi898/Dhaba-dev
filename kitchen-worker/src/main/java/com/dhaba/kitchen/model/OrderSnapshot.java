package com.dhaba.kitchen.model;

import java.time.Instant;
import java.util.List;

/** Read-only view of an order loaded from PostgreSQL. */
public record OrderSnapshot(
        long id,
        String customerName,
        Integer tableNumber,
        OrderStatus status,
        int totalCents,
        Instant createdAt,
        List<OrderLine> lines) {

    public OrderSnapshot {
        lines = List.copyOf(lines);
    }

    public int itemCount() {
        return lines.stream().mapToInt(OrderLine::quantity).sum();
    }
}
