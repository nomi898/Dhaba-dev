package com.dhaba.kitchen.stats;

import com.dhaba.kitchen.config.RedisKeys;
import com.dhaba.kitchen.model.OrderLine;
import com.dhaba.kitchen.model.OrderSnapshot;
import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.queue.JobCodec;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.data.redis.core.script.RedisScript;
import org.springframework.stereotype.Component;

/** Redis implementation of {@link StatsRecorder}. All updates are atomic Lua scripts. */
@Component
public class RedisStatsRecorder implements StatsRecorder {

    static final DateTimeFormatter MINUTE = DateTimeFormatter.ofPattern("yyyyMMddHHmm").withZone(ZoneOffset.UTC);
    private static final long MINUTE_BUCKET_TTL_SECONDS = 3 * 60 * 60;
    private static final long COUNTED_TTL_SECONDS = 7 * 24 * 60 * 60;
    private static final long ORDER_STATUS_TTL_SECONDS = 24 * 60 * 60;
    private static final int RECENT_LIMIT = 50;

    private static final RedisScript<Long> RECORD_ORDER = new DefaultRedisScript<>("""
            if not redis.call('SET', KEYS[1], '1', 'NX', 'EX', ARGV[4]) then return 0 end
            redis.call('INCR', KEYS[2])
            redis.call('INCRBY', KEYS[3], ARGV[1])
            redis.call('INCR', KEYS[4])
            redis.call('EXPIRE', KEYS[4], ARGV[3])
            redis.call('LPUSH', KEYS[8], ARGV[2])
            redis.call('LTRIM', KEYS[8], 0, tonumber(ARGV[7]) - 1)
            redis.call('HINCRBY', KEYS[9], ARGV[5], 1)
            redis.call('SET', KEYS[10], ARGV[5], 'EX', ARGV[6])
            for i = 8, #ARGV, 4 do
              redis.call('ZINCRBY', KEYS[5], ARGV[i + 1], ARGV[i])
              redis.call('ZINCRBY', KEYS[6], ARGV[i + 2], ARGV[i])
              redis.call('HSET', KEYS[7], ARGV[i], ARGV[i + 3])
            end
            return 1""", Long.class);

    private static final RedisScript<Long> RECORD_TRANSITION = new DefaultRedisScript<>("""
            local current = tonumber(redis.call('HGET', KEYS[1], ARGV[1]) or '0')
            if current > 0 then redis.call('HINCRBY', KEYS[1], ARGV[1], -1) end
            redis.call('HINCRBY', KEYS[1], ARGV[2], 1)
            redis.call('SET', KEYS[2], ARGV[2], 'EX', ARGV[3])
            return 1""", Long.class);

    private final StringRedisTemplate redis;
    private final JobCodec codec;

    public RedisStatsRecorder(StringRedisTemplate redis, JobCodec codec) {
        this.redis = redis;
        this.codec = codec;
    }

    @Override
    public boolean recordOrder(OrderSnapshot order) {
        List<String> keys = List.of(
                RedisKeys.STATS_COUNTED_PREFIX + order.id(),
                RedisKeys.STATS_TOTAL,
                RedisKeys.STATS_REVENUE,
                RedisKeys.STATS_MINUTE_PREFIX + MINUTE.format(order.createdAt()),
                RedisKeys.STATS_DISH_QTY,
                RedisKeys.STATS_DISH_REVENUE,
                RedisKeys.STATS_DISH_NAMES,
                RedisKeys.STATS_RECENT,
                RedisKeys.STATS_STATUS_COUNTS,
                RedisKeys.orderStatus(order.id()));

        List<String> args = new ArrayList<>();
        args.add(String.valueOf(order.totalCents()));
        args.add(codec.encode(recentEntry(order)));
        args.add(String.valueOf(MINUTE_BUCKET_TTL_SECONDS));
        args.add(String.valueOf(COUNTED_TTL_SECONDS));
        args.add(order.status().name());
        args.add(String.valueOf(ORDER_STATUS_TTL_SECONDS));
        args.add(String.valueOf(RECENT_LIMIT));
        for (OrderLine line : order.lines()) {
            args.add(String.valueOf(line.menuItemId()));
            args.add(String.valueOf(line.quantity()));
            args.add(String.valueOf(line.lineTotalCents()));
            args.add(line.name());
        }
        Long result = redis.execute(RECORD_ORDER, keys, args.toArray());
        return result != null && result == 1L;
    }

    @Override
    public void recordTransition(long orderId, OrderStatus from, OrderStatus to) {
        redis.execute(RECORD_TRANSITION, List.of(RedisKeys.STATS_STATUS_COUNTS, RedisKeys.orderStatus(orderId)),
                from.name(), to.name(), String.valueOf(ORDER_STATUS_TTL_SECONDS));
    }

    static Map<String, Object> recentEntry(OrderSnapshot order) {
        Map<String, Object> entry = new LinkedHashMap<>();
        entry.put("id", order.id());
        entry.put("customerName", order.customerName());
        entry.put("tableNumber", order.tableNumber());
        entry.put("totalCents", order.totalCents());
        entry.put("itemCount", order.itemCount());
        entry.put("createdAt", order.createdAt().toString());
        return entry;
    }
}
