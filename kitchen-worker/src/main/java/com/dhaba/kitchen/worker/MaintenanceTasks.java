package com.dhaba.kitchen.worker;

import com.dhaba.kitchen.queue.JobQueue;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

/** Periodic queue housekeeping: promote due delayed jobs and recover jobs from dead workers. */
@Component
public class MaintenanceTasks {

    private static final Logger LOG = LoggerFactory.getLogger(MaintenanceTasks.class);

    private final JobQueue queue;

    public MaintenanceTasks(JobQueue queue) {
        this.queue = queue;
    }

    @Scheduled(fixedDelay = 1000, initialDelay = 1000)
    public void promoteDelayed() {
        try {
            long n = queue.promoteDue();
            if (n > 0) {
                LOG.debug("promoted {} delayed job(s)", n);
            }
        } catch (DataAccessException e) {
            LOG.warn("promote delayed jobs failed: {}", e.getMessage());
        }
    }

    @Scheduled(fixedDelay = 30_000, initialDelay = 15_000)
    public void recoverOrphans() {
        try {
            queue.recoverOrphans();
        } catch (DataAccessException e) {
            LOG.warn("orphan recovery failed: {}", e.getMessage());
        }
    }
}
