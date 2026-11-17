package com.dhaba.kitchen.model;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class OrderStatusTest {

    @Test
    void lifecycleMovesForwardOnly() {
        assertThat(OrderStatus.NEW.next()).contains(OrderStatus.COOKING);
        assertThat(OrderStatus.COOKING.next()).contains(OrderStatus.READY);
        assertThat(OrderStatus.READY.next()).isEmpty();
        assertThat(OrderStatus.READY.previous()).contains(OrderStatus.COOKING);
        assertThat(OrderStatus.NEW.previous()).isEmpty();
    }

    @Test
    void ordering() {
        assertThat(OrderStatus.READY.isAtOrAfter(OrderStatus.COOKING)).isTrue();
        assertThat(OrderStatus.COOKING.isAtOrAfter(OrderStatus.COOKING)).isTrue();
        assertThat(OrderStatus.NEW.isAtOrAfter(OrderStatus.COOKING)).isFalse();
    }

    @Test
    void nextAttemptKeepsIdentity() {
        Job job = Job.statusChange(7, OrderStatus.READY, "kitchen");
        Job retry = job.nextAttempt();
        assertThat(retry.id()).isEqualTo(job.id());
        assertThat(retry.attempt()).isEqualTo(1);
        assertThat(retry.targetStatus()).isEqualTo(OrderStatus.READY);
    }
}
