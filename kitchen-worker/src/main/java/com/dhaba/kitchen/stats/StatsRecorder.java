package com.dhaba.kitchen.stats;

import com.dhaba.kitchen.model.OrderSnapshot;
import com.dhaba.kitchen.model.OrderStatus;

/** Writes live statistics that stats-api reads. */
public interface StatsRecorder {

    /** Count a new order once. Returns false if it was already counted. */
    boolean recordOrder(OrderSnapshot order);

    void recordTransition(long orderId, OrderStatus from, OrderStatus to);
}
