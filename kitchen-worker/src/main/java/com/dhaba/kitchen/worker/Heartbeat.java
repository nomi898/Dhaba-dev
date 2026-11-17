package com.dhaba.kitchen.worker;

import com.dhaba.kitchen.config.RedisKeys;
import com.dhaba.kitchen.config.WorkerProperties;
import com.dhaba.kitchen.queue.JobCodec;
import jakarta.annotation.PreDestroy;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

/**
 * Writes a heartbeat to Redis every {@code worker.heartbeat-interval} (10s by default).
 *
 * <p>Keys: {@code kitchen-worker:heartbeat} (latest heartbeat from any worker) and
 * {@code kitchen-worker:heartbeat:<id>} (per worker). Both expire after {@code heartbeat-ttl}, so a missing key
 * means no live worker. The same timestamp is written to a local file for container liveness probes.
 */
@Component
public class Heartbeat {

    private static final Logger LOG = LoggerFactory.getLogger(Heartbeat.class);

    private final StringRedisTemplate redis;
    private final JobCodec codec;
    private final WorkerProperties props;
    private final WorkerMetrics metrics;
    private final Instant startedAt = Instant.now();

    public Heartbeat(StringRedisTemplate redis, JobCodec codec, WorkerProperties props, WorkerMetrics metrics) {
        this.redis = redis;
        this.codec = codec;
        this.props = props;
        this.metrics = metrics;
    }

    @Scheduled(fixedRateString = "${worker.heartbeat-interval}")
    public void beat() {
        Instant now = Instant.now();
        String payload = codec.encode(payload(now));
        try {
            redis.opsForValue().set(RedisKeys.heartbeat(props.id()), payload, props.heartbeatTtl());
            redis.opsForValue().set(RedisKeys.HEARTBEAT, payload, props.heartbeatTtl());
            redis.opsForSet().add(RedisKeys.WORKERS, props.id());
            writeFile(now);
            LOG.debug("heartbeat {}", payload);
        } catch (DataAccessException e) {
            LOG.error("heartbeat failed: {}", e.getMessage());
        }
    }

    Map<String, Object> payload(Instant now) {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("workerId", props.id());
        body.put("version", props.version());
        body.put("status", "running");
        body.put("timestamp", now.toString());
        body.put("startedAt", startedAt.toString());
        body.put("processed", metrics.processedCount());
        body.put("retried", metrics.retriedCount());
        body.put("deadLettered", metrics.deadLetteredCount());
        Instant last = metrics.lastJobAt();
        body.put("lastJobAt", last == null ? null : last.toString());
        return body;
    }

    private void writeFile(Instant now) {
        if (props.heartbeatFile() == null || props.heartbeatFile().isBlank()) {
            return;
        }
        try {
            Files.writeString(Path.of(props.heartbeatFile()), now.toString());
        } catch (IOException e) {
            LOG.warn("cannot write heartbeat file {}: {}", props.heartbeatFile(), e.getMessage());
        }
    }

    @PreDestroy
    public void clear() {
        try {
            redis.delete(RedisKeys.heartbeat(props.id()));
            redis.opsForSet().remove(RedisKeys.WORKERS, props.id());
            LOG.info("heartbeat cleared for {}", props.id());
        } catch (DataAccessException e) {
            LOG.warn("could not clear heartbeat: {}", e.getMessage());
        }
    }
}
