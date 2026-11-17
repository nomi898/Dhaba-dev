package com.dhaba.kitchen.worker;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.dhaba.kitchen.TestFixtures;
import com.dhaba.kitchen.config.RedisKeys;
import com.dhaba.kitchen.queue.JobCodec;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.data.redis.core.SetOperations;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

class HeartbeatTest {

    @Test
    @SuppressWarnings("unchecked")
    void writesHeartbeatKeysWithTtl() {
        StringRedisTemplate redis = mock(StringRedisTemplate.class);
        ValueOperations<String, String> values = mock(ValueOperations.class);
        SetOperations<String, String> sets = mock(SetOperations.class);
        when(redis.opsForValue()).thenReturn(values);
        when(redis.opsForSet()).thenReturn(sets);

        Heartbeat heartbeat = new Heartbeat(redis, new JobCodec(new ObjectMapper()), TestFixtures.props(false),
                new WorkerMetrics());
        heartbeat.beat();

        verify(values).set(eq(RedisKeys.heartbeat("worker-test")), anyString(), eq(Duration.ofSeconds(30)));
        verify(values).set(eq(RedisKeys.HEARTBEAT), anyString(), eq(Duration.ofSeconds(30)));
        verify(sets).add(RedisKeys.WORKERS, "worker-test");
    }

    @Test
    void payloadDescribesWorker() {
        WorkerMetrics metrics = new WorkerMetrics();
        metrics.processed();
        Heartbeat heartbeat = new Heartbeat(mock(StringRedisTemplate.class), new JobCodec(new ObjectMapper()),
                TestFixtures.props(false), metrics);
        Map<String, Object> body = heartbeat.payload(Instant.parse("2026-11-17T10:00:00Z"));
        assertThat(body).containsEntry("workerId", "worker-test").containsEntry("status", "running")
                .containsEntry("timestamp", "2026-11-17T10:00:00Z").containsEntry("processed", 1L);
        assertThat(body.get("lastJobAt")).isNotNull();
    }
}
