package com.dhaba.kitchen.stats;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.dhaba.kitchen.TestFixtures;
import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.queue.JobCodec;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.RedisScript;

class RedisStatsRecorderTest {

    @Test
    void minuteBucketUsesUtc() {
        assertThat(RedisStatsRecorder.MINUTE.format(TestFixtures.order(1, OrderStatus.NEW).createdAt()))
                .isEqualTo("202611171015");
    }

    @Test
    void recentEntryContainsSummary() {
        Map<String, Object> entry = RedisStatsRecorder.recentEntry(TestFixtures.order(9, OrderStatus.NEW));
        assertThat(entry).containsEntry("id", 9L).containsEntry("itemCount", 3).containsEntry("totalCents", 169000);
    }

    @Test
    @SuppressWarnings({"unchecked", "rawtypes"})
    void recordOrderPassesKeysAndDishArguments() {
        StringRedisTemplate redis = mock(StringRedisTemplate.class);
        when(redis.execute(any(RedisScript.class), anyList(), any(Object[].class))).thenReturn(1L);
        RedisStatsRecorder recorder = new RedisStatsRecorder(redis, new JobCodec(new ObjectMapper()));

        assertThat(recorder.recordOrder(TestFixtures.order(9, OrderStatus.NEW))).isTrue();

        ArgumentCaptor<List> keys = ArgumentCaptor.forClass(List.class);
        ArgumentCaptor<Object[]> args = ArgumentCaptor.forClass(Object[].class);
        verify(redis).execute(any(RedisScript.class), keys.capture(), args.capture());
        assertThat(keys.getValue()).contains("stats:counted:9", "stats:orders:minute:202611171015",
                "stats:order:9:status");
        assertThat(args.getValue()).hasSize(7 + 8).contains("Chicken Karahi", "Garlic Naan", "NEW");
    }
}
