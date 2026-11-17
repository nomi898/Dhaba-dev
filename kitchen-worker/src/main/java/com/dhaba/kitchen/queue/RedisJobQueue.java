package com.dhaba.kitchen.queue;

import com.dhaba.kitchen.config.RedisKeys;
import com.dhaba.kitchen.config.WorkerProperties;
import com.dhaba.kitchen.model.Job;
import java.time.Duration;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.connection.RedisListCommands.Direction;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.data.redis.core.script.RedisScript;
import org.springframework.stereotype.Component;

/**
 * Redis-backed reliable queue.
 *
 * <p>Jobs are atomically moved from {@code orders:queue} into a per-worker processing list with BLMOVE, so a
 * crash never loses a job: a peer worker returns it once the crashed worker's heartbeat expires.
 */
@Component
public class RedisJobQueue implements JobQueue {

    private static final Logger LOG = LoggerFactory.getLogger(RedisJobQueue.class);
    private static final int DEAD_LETTER_LIMIT = 1000;

    private static final RedisScript<Long> RETRY = new DefaultRedisScript<>("""
            redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
            return redis.call('LREM', KEYS[2], 1, ARGV[3])""", Long.class);

    private static final RedisScript<Long> DEAD_LETTER = new DefaultRedisScript<>("""
            redis.call('LPUSH', KEYS[1], ARGV[1])
            redis.call('LTRIM', KEYS[1], 0, tonumber(ARGV[3]) - 1)
            return redis.call('LREM', KEYS[2], 1, ARGV[2])""", Long.class);

    private static final RedisScript<Long> PROMOTE = new DefaultRedisScript<>("""
            local due = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 100)
            for _, job in ipairs(due) do
              redis.call('ZREM', KEYS[1], job)
              redis.call('LPUSH', KEYS[2], job)
            end
            return #due""", Long.class);

    /** KEYS: heartbeat, processing, queue, workers. ARGV: workerId, force. */
    private static final RedisScript<Long> RECOVER = new DefaultRedisScript<>("""
            if ARGV[2] ~= '1' and redis.call('EXISTS', KEYS[1]) == 1 then return -1 end
            local n = 0
            while redis.call('LMOVE', KEYS[2], KEYS[3], 'RIGHT', 'RIGHT') do n = n + 1 end
            if ARGV[2] ~= '1' then redis.call('SREM', KEYS[4], ARGV[1]) end
            return n""", Long.class);

    private final StringRedisTemplate redis;
    private final JobCodec codec;
    private final String workerId;
    private final String processingKey;

    public RedisJobQueue(StringRedisTemplate redis, JobCodec codec, WorkerProperties props) {
        this.redis = redis;
        this.codec = codec;
        this.workerId = props.id();
        this.processingKey = RedisKeys.processing(props.id());
    }

    @Override
    public Optional<String> take(Duration timeout) {
        String raw = redis.opsForList().move(RedisKeys.QUEUE, Direction.RIGHT, processingKey, Direction.LEFT,
                timeout);
        return Optional.ofNullable(raw);
    }

    @Override
    public void ack(String raw) {
        redis.opsForList().remove(processingKey, 1, raw);
    }

    @Override
    public void retry(String raw, Job next, Duration delay) {
        long runAt = Instant.now().plus(delay).toEpochMilli();
        redis.execute(RETRY, List.of(RedisKeys.DELAYED, processingKey),
                String.valueOf(runAt), codec.encode(next), raw);
    }

    @Override
    public void deadLetter(String raw, String reason) {
        Map<String, Object> envelope = new LinkedHashMap<>();
        envelope.put("job", raw);
        envelope.put("reason", reason);
        envelope.put("workerId", workerId);
        envelope.put("failedAt", Instant.now().toString());
        redis.execute(DEAD_LETTER, List.of(RedisKeys.DEAD, processingKey),
                codec.encode(envelope), raw, String.valueOf(DEAD_LETTER_LIMIT));
    }

    @Override
    public void schedule(Job job, Duration delay) {
        long runAt = Instant.now().plus(delay).toEpochMilli();
        redis.opsForZSet().add(RedisKeys.DELAYED, codec.encode(job), runAt);
    }

    @Override
    public long promoteDue() {
        Long moved = redis.execute(PROMOTE, List.of(RedisKeys.DELAYED, RedisKeys.QUEUE),
                String.valueOf(Instant.now().toEpochMilli()));
        return moved == null ? 0 : moved;
    }

    @Override
    public long recoverOrphans() {
        Set<String> workers = redis.opsForSet().members(RedisKeys.WORKERS);
        if (workers == null) {
            return 0;
        }
        long total = 0;
        for (String other : workers) {
            if (other.equals(workerId)) {
                continue;
            }
            long n = recover(other, false);
            if (n >= 0) {
                LOG.warn("worker {} is gone; returned {} in-flight job(s) to the queue", other, n);
                total += n;
            }
        }
        return total;
    }

    @Override
    public long recoverOwn() {
        long n = recover(workerId, true);
        if (n > 0) {
            LOG.warn("returned {} job(s) left over from a previous run to the queue", n);
        }
        return Math.max(n, 0);
    }

    private long recover(String id, boolean force) {
        Long n = redis.execute(RECOVER,
                List.of(RedisKeys.heartbeat(id), RedisKeys.processing(id), RedisKeys.QUEUE, RedisKeys.WORKERS),
                id, force ? "1" : "0");
        return n == null ? 0 : n;
    }
}
