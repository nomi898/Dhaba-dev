package com.dhaba.kitchen.queue;

import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.PermanentJobException;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;

/** Converts jobs to and from their JSON wire format. */
@Component
public class JobCodec {

    private final ObjectMapper mapper;

    public JobCodec(ObjectMapper mapper) {
        this.mapper = mapper;
    }

    public Job decode(String raw) {
        try {
            Job job = mapper.readValue(raw, Job.class);
            if (job.type() == null || job.orderId() <= 0) {
                throw new PermanentJobException("job is missing type or orderId: " + raw);
            }
            return job;
        } catch (JsonProcessingException e) {
            throw new PermanentJobException("unparseable job: " + e.getOriginalMessage());
        }
    }

    public String encode(Object value) {
        try {
            return mapper.writeValueAsString(value);
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("cannot serialize " + value.getClass().getSimpleName(), e);
        }
    }
}
