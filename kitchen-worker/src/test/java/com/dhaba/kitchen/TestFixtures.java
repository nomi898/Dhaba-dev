package com.dhaba.kitchen;

import com.dhaba.kitchen.config.WorkerProperties;
import com.dhaba.kitchen.model.OrderLine;
import com.dhaba.kitchen.model.OrderSnapshot;
import com.dhaba.kitchen.model.OrderStatus;
import java.time.Duration;
import java.time.Instant;
import java.util.List;

public final class TestFixtures {

    private TestFixtures() {
    }

    public static WorkerProperties props(boolean autoAdvance) {
        return new WorkerProperties("worker-test", "test", 3, Duration.ofSeconds(1), Duration.ofSeconds(10),
                Duration.ofMillis(100), Duration.ofSeconds(10), Duration.ofSeconds(30), "", autoAdvance,
                Duration.ofSeconds(5), Duration.ofSeconds(20));
    }

    public static OrderSnapshot order(long id, OrderStatus status) {
        return new OrderSnapshot(id, "Ali", 4, status, 169000, Instant.parse("2026-11-17T10:15:30Z"),
                List.of(new OrderLine(4, "Chicken Karahi", 1, 145000), new OrderLine(15, "Garlic Naan", 2, 24000)));
    }
}
