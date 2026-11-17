package com.dhaba.kitchen.queue;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.JobType;
import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.model.PermanentJobException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.junit.jupiter.api.Test;

class JobCodecTest {

    private final JobCodec codec = new JobCodec(new ObjectMapper().registerModule(new JavaTimeModule())
            .disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS));

    @Test
    void decodesJobWrittenByOrdersApi() {
        String raw = "{\"id\":\"3f6c\",\"type\":\"ORDER_CREATED\",\"orderId\":42,\"source\":\"orders-api\","
                + "\"attempt\":0,\"createdAt\":\"2026-11-17T10:15:30.123456789Z\",\"futureField\":true}";
        Job job = codec.decode(raw);
        assertThat(job.type()).isEqualTo(JobType.ORDER_CREATED);
        assertThat(job.orderId()).isEqualTo(42);
        assertThat(job.targetStatus()).isNull();
    }

    @Test
    void roundTripsStatusChange() {
        Job job = Job.statusChange(9, OrderStatus.COOKING, "kitchen");
        Job back = codec.decode(codec.encode(job));
        assertThat(back).isEqualTo(job);
        assertThat(codec.encode(job)).contains("\"targetStatus\":\"COOKING\"").contains("\"createdAt\":\"20");
    }

    @Test
    void rejectsMalformedJobs() {
        assertThatThrownBy(() -> codec.decode("not json")).isInstanceOf(PermanentJobException.class);
        assertThatThrownBy(() -> codec.decode("{\"type\":\"ORDER_CREATED\"}"))
                .isInstanceOf(PermanentJobException.class);
        assertThatThrownBy(() -> codec.decode("{\"type\":\"EXPLODE\",\"orderId\":1}"))
                .isInstanceOf(PermanentJobException.class);
    }
}
