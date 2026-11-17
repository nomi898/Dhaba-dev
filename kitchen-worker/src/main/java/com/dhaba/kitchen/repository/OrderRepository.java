package com.dhaba.kitchen.repository;

import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.model.OrderSnapshot;
import java.util.Optional;

/** Order persistence used by the worker. */
public interface OrderRepository {

    Optional<OrderSnapshot> findOrder(long orderId);

    /**
     * Atomically move an order from {@code from} to {@code to}.
     *
     * @return true if the row was updated, false if the order was not in {@code from}
     */
    boolean transition(long orderId, OrderStatus from, OrderStatus to);
}
