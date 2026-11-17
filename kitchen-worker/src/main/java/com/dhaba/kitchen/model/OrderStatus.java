package com.dhaba.kitchen.model;

import java.util.Optional;

/** Order lifecycle: NEW -> COOKING -> READY. */
public enum OrderStatus {
    NEW, COOKING, READY;

    public Optional<OrderStatus> next() {
        return switch (this) {
            case NEW -> Optional.of(COOKING);
            case COOKING -> Optional.of(READY);
            case READY -> Optional.empty();
        };
    }

    public Optional<OrderStatus> previous() {
        return switch (this) {
            case NEW -> Optional.empty();
            case COOKING -> Optional.of(NEW);
            case READY -> Optional.of(COOKING);
        };
    }

    /** True if this status is the same as, or later than, {@code other}. */
    public boolean isAtOrAfter(OrderStatus other) {
        return compareTo(other) >= 0;
    }
}
