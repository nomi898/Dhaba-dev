package com.dhaba.kitchen.config;

/** Redis key contract shared with orders-api and stats-api. */
public final class RedisKeys {

    public static final String QUEUE = "orders:queue";
    public static final String PROCESSING_PREFIX = "orders:processing:";
    public static final String DELAYED = "orders:delayed";
    public static final String DEAD = "orders:dead";

    public static final String HEARTBEAT = "kitchen-worker:heartbeat";
    public static final String HEARTBEAT_PREFIX = "kitchen-worker:heartbeat:";
    public static final String WORKERS = "kitchen-worker:workers";

    public static final String STATS_TOTAL = "stats:orders:total";
    public static final String STATS_REVENUE = "stats:revenue:total_cents";
    public static final String STATS_MINUTE_PREFIX = "stats:orders:minute:";
    public static final String STATS_DISH_QTY = "stats:dishes:qty";
    public static final String STATS_DISH_REVENUE = "stats:dishes:revenue";
    public static final String STATS_DISH_NAMES = "stats:dishes:names";
    public static final String STATS_RECENT = "stats:recent";
    public static final String STATS_STATUS_COUNTS = "stats:status_counts";
    public static final String STATS_ORDER_STATUS_PREFIX = "stats:order:";
    public static final String STATS_COUNTED_PREFIX = "stats:counted:";

    private RedisKeys() {
    }

    public static String processing(String workerId) {
        return PROCESSING_PREFIX + workerId;
    }

    public static String heartbeat(String workerId) {
        return HEARTBEAT_PREFIX + workerId;
    }

    public static String orderStatus(long orderId) {
        return STATS_ORDER_STATUS_PREFIX + orderId + ":status";
    }
}
