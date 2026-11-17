package com.dhaba.kitchen.repository;

import com.dhaba.kitchen.model.OrderLine;
import com.dhaba.kitchen.model.OrderSnapshot;
import com.dhaba.kitchen.model.OrderStatus;
import java.sql.Timestamp;
import java.util.List;
import java.util.Optional;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

/** JDBC implementation of {@link OrderRepository}. */
@Repository
public class JdbcOrderRepository implements OrderRepository {

    private static final String ORDER_SQL = """
            SELECT id, customer_name, table_number, status, total_cents, created_at
            FROM orders WHERE id = ?""";

    private static final String LINES_SQL = """
            SELECT menu_item_id, item_name, quantity, line_total_cents
            FROM order_items WHERE order_id = ? ORDER BY id""";

    private static final String TRANSITION_SQL = """
            UPDATE orders
               SET status = ?,
                   cooking_started_at = CASE WHEN ? = 'COOKING' THEN now() ELSE cooking_started_at END,
                   ready_at = CASE WHEN ? = 'READY' THEN now() ELSE ready_at END
             WHERE id = ? AND status = ?""";

    private final JdbcTemplate jdbc;

    public JdbcOrderRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<OrderSnapshot> findOrder(long orderId) {
        List<OrderSnapshot> found = jdbc.query(ORDER_SQL, (rs, n) -> {
            Timestamp created = rs.getTimestamp("created_at");
            int table = rs.getInt("table_number");
            Integer tableNumber = rs.wasNull() ? null : table;
            return new OrderSnapshot(rs.getLong("id"), rs.getString("customer_name"), tableNumber,
                    OrderStatus.valueOf(rs.getString("status")), rs.getInt("total_cents"),
                    created.toInstant(), List.of());
        }, orderId);
        if (found.isEmpty()) {
            return Optional.empty();
        }
        List<OrderLine> lines = jdbc.query(LINES_SQL, (rs, n) -> new OrderLine(
                rs.getInt("menu_item_id"), rs.getString("item_name"),
                rs.getInt("quantity"), rs.getInt("line_total_cents")), orderId);
        OrderSnapshot o = found.get(0);
        return Optional.of(new OrderSnapshot(o.id(), o.customerName(), o.tableNumber(), o.status(),
                o.totalCents(), o.createdAt(), lines));
    }

    @Override
    public boolean transition(long orderId, OrderStatus from, OrderStatus to) {
        int updated = jdbc.update(TRANSITION_SQL, to.name(), to.name(), to.name(), orderId, from.name());
        return updated == 1;
    }
}
