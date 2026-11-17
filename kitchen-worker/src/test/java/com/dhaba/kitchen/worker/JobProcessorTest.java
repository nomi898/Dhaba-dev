package com.dhaba.kitchen.worker;

import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import com.dhaba.kitchen.TestFixtures;
import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.JobType;
import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.model.PermanentJobException;
import com.dhaba.kitchen.queue.JobQueue;
import com.dhaba.kitchen.repository.OrderRepository;
import com.dhaba.kitchen.stats.StatsRecorder;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

class JobProcessorTest {

    private OrderRepository orders;
    private StatsRecorder stats;
    private JobQueue queue;

    @BeforeEach
    void setUp() {
        orders = mock(OrderRepository.class);
        stats = mock(StatsRecorder.class);
        queue = mock(JobQueue.class);
    }

    private JobProcessor processor(boolean autoAdvance) {
        return new JobProcessor(orders, stats, queue, TestFixtures.props(autoAdvance));
    }

    private static Job created(long id) {
        return new Job("j1", JobType.ORDER_CREATED, id, null, "orders-api", 0, Instant.now());
    }

    @Test
    void orderCreatedRecordsStatsAndSchedulesCookingWhenAutoAdvance() {
        when(orders.findOrder(1)).thenReturn(Optional.of(TestFixtures.order(1, OrderStatus.NEW)));
        processor(true).process(created(1));
        verify(stats).recordOrder(any());
        verify(queue).schedule(argThat(j -> j.type() == JobType.STATUS_CHANGE
                && j.targetStatus() == OrderStatus.COOKING && j.orderId() == 1), any(Duration.class));
    }

    @Test
    void orderCreatedWithoutAutoAdvanceOnlyRecordsStats() {
        when(orders.findOrder(1)).thenReturn(Optional.of(TestFixtures.order(1, OrderStatus.NEW)));
        processor(false).process(created(1));
        verify(stats).recordOrder(any());
        verifyNoInteractions(queue);
    }

    @Test
    void missingOrderIsPermanentFailure() {
        when(orders.findOrder(5)).thenReturn(Optional.empty());
        assertThatThrownBy(() -> processor(true).process(created(5))).isInstanceOf(PermanentJobException.class);
    }

    @Test
    void statusChangeAppliesTransitionAndRecordsStats() {
        when(orders.findOrder(2)).thenReturn(Optional.of(TestFixtures.order(2, OrderStatus.NEW)));
        when(orders.transition(2, OrderStatus.NEW, OrderStatus.COOKING)).thenReturn(true);
        processor(true).process(Job.statusChange(2, OrderStatus.COOKING, "kitchen"));
        verify(stats).recordTransition(2, OrderStatus.NEW, OrderStatus.COOKING);
        verify(queue).schedule(argThat(j -> j.targetStatus() == OrderStatus.READY), any(Duration.class));
    }

    @Test
    void readyTransitionSchedulesNothing() {
        when(orders.findOrder(2)).thenReturn(Optional.of(TestFixtures.order(2, OrderStatus.COOKING)));
        when(orders.transition(2, OrderStatus.COOKING, OrderStatus.READY)).thenReturn(true);
        processor(true).process(Job.statusChange(2, OrderStatus.READY, "kitchen"));
        verify(stats).recordTransition(2, OrderStatus.COOKING, OrderStatus.READY);
        verifyNoInteractions(queue);
    }

    @Test
    void duplicateOrStaleStatusChangeIsIgnored() {
        when(orders.findOrder(3)).thenReturn(Optional.of(TestFixtures.order(3, OrderStatus.READY)));
        processor(true).process(Job.statusChange(3, OrderStatus.COOKING, "kitchen-worker"));
        verify(orders, never()).transition(anyLong(), any(), any());
        verifyNoInteractions(stats, queue);
    }

    @Test
    void concurrentUpdateIsTreatedAsStale() {
        when(orders.findOrder(4)).thenReturn(Optional.of(TestFixtures.order(4, OrderStatus.NEW)));
        when(orders.transition(4, OrderStatus.NEW, OrderStatus.COOKING)).thenReturn(false);
        processor(true).process(Job.statusChange(4, OrderStatus.COOKING, "kitchen"));
        verifyNoInteractions(stats, queue);
    }

    @Test
    void skippingAStateIsPermanentFailure() {
        when(orders.findOrder(6)).thenReturn(Optional.of(TestFixtures.order(6, OrderStatus.NEW)));
        assertThatThrownBy(() -> processor(true).process(Job.statusChange(6, OrderStatus.READY, "kitchen")))
                .isInstanceOf(PermanentJobException.class);
    }

    @Test
    void invalidTargetsArePermanentFailures() {
        Job noTarget = new Job("j", JobType.STATUS_CHANGE, 1, null, "x", 0, Instant.now());
        assertThatThrownBy(() -> processor(true).process(noTarget)).isInstanceOf(PermanentJobException.class);
        assertThatThrownBy(() -> processor(true).process(Job.statusChange(1, OrderStatus.NEW, "x")))
                .isInstanceOf(PermanentJobException.class);
    }
}
