package com.dhaba.kitchen.worker;

import com.dhaba.kitchen.config.WorkerProperties;
import com.dhaba.kitchen.model.Job;
import com.dhaba.kitchen.model.OrderSnapshot;
import com.dhaba.kitchen.model.OrderStatus;
import com.dhaba.kitchen.model.PermanentJobException;
import com.dhaba.kitchen.queue.JobQueue;
import com.dhaba.kitchen.repository.OrderRepository;
import com.dhaba.kitchen.stats.StatsRecorder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

/**
 * Business logic for a single job. Every handler is idempotent, so a job that is delivered twice
 * (after a crash or a retry) has no extra effect.
 */
@Service
public class JobProcessor {

    private static final Logger LOG = LoggerFactory.getLogger(JobProcessor.class);
    static final String AUTO_SOURCE = "kitchen-worker";

    private final OrderRepository orders;
    private final StatsRecorder stats;
    private final JobQueue queue;
    private final WorkerProperties props;

    public JobProcessor(OrderRepository orders, StatsRecorder stats, JobQueue queue, WorkerProperties props) {
        this.orders = orders;
        this.stats = stats;
        this.queue = queue;
        this.props = props;
    }

    public void process(Job job) {
        switch (job.type()) {
            case ORDER_CREATED -> handleCreated(job);
            case STATUS_CHANGE -> handleStatusChange(job);
            default -> throw new PermanentJobException("unsupported job type " + job.type());
        }
    }

    private OrderSnapshot load(long orderId) {
        return orders.findOrder(orderId)
                .orElseThrow(() -> new PermanentJobException("order " + orderId + " does not exist"));
    }

    private void handleCreated(Job job) {
        OrderSnapshot order = load(job.orderId());
        boolean counted = stats.recordOrder(order);
        LOG.info("order {} received: {} item(s), total {} ({})", order.id(), order.itemCount(), order.totalCents(),
                counted ? "stats recorded" : "already counted");
        if (props.autoAdvance() && order.status() == OrderStatus.NEW) {
            queue.schedule(Job.statusChange(order.id(), OrderStatus.COOKING, AUTO_SOURCE), props.acceptDelay());
        }
    }

    private void handleStatusChange(Job job) {
        OrderStatus target = job.targetStatus();
        if (target == null) {
            throw new PermanentJobException("STATUS_CHANGE job " + job.id() + " has no targetStatus");
        }
        OrderStatus from = target.previous()
                .orElseThrow(() -> new PermanentJobException("cannot transition an order into " + target));
        OrderSnapshot order = load(job.orderId());

        if (order.status().isAtOrAfter(target)) {
            LOG.info("order {} already {}; ignoring {} request from {}", order.id(), order.status(), target,
                    job.source());
            return;
        }
        if (order.status() != from) {
            throw new PermanentJobException("order " + order.id() + " is " + order.status() + ", cannot move to "
                    + target);
        }
        if (!orders.transition(order.id(), from, target)) {
            LOG.info("order {} changed concurrently; {} request is stale", order.id(), target);
            return;
        }
        stats.recordTransition(order.id(), from, target);
        LOG.info("order {} {} -> {} (requested by {})", order.id(), from, target, job.source());

        if (props.autoAdvance() && target == OrderStatus.COOKING) {
            queue.schedule(Job.statusChange(order.id(), OrderStatus.READY, AUTO_SOURCE), props.cookTime());
        }
    }
}
