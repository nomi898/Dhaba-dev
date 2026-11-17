// Package store implements order persistence on PostgreSQL.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dhaba-orders/orders-api/internal/order"
)

// Postgres is an order.Store backed by pgx.
type Postgres struct {
	pool *pgxpool.Pool
}

// New connects a pool to PostgreSQL.
func New(ctx context.Context, url string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse postgres url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// Close releases the pool.
func (p *Postgres) Close() { p.pool.Close() }

// Ping checks connectivity.
func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

// MenuItems loads menu items by id.
func (p *Postgres) MenuItems(ctx context.Context, ids []int) (map[int]order.MenuItem, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, name, price_cents, is_available FROM menu_items WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int]order.MenuItem, len(ids))
	for rows.Next() {
		var m order.MenuItem
		if err := rows.Scan(&m.ID, &m.Name, &m.PriceCents, &m.IsAvailable); err != nil {
			return nil, err
		}
		out[m.ID] = m
	}
	return out, rows.Err()
}

// CreateOrder inserts the order and its items in one transaction.
func (p *Postgres) CreateOrder(ctx context.Context, o order.NewOrder) (order.Order, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return order.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := order.Order{
		CustomerName: o.CustomerName, TableNumber: o.TableNumber, Notes: o.Notes,
		TotalCents: o.TotalCents, Items: o.Items,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer_name, table_number, notes, total_cents)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at, updated_at`,
		o.CustomerName, o.TableNumber, o.Notes, o.TotalCents,
	).Scan(&created.ID, &created.Status, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return order.Order{}, fmt.Errorf("insert order: %w", err)
	}

	batch := &pgx.Batch{}
	for _, it := range o.Items {
		batch.Queue(`
			INSERT INTO order_items (order_id, menu_item_id, item_name, unit_price_cents, quantity, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			created.ID, it.MenuItemID, it.Name, it.UnitPriceCents, it.Quantity, it.LineTotalCents)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return order.Order{}, fmt.Errorf("insert order items: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return order.Order{}, fmt.Errorf("commit: %w", err)
	}
	return created, nil
}

// MarkEnqueued records that the ORDER_CREATED job was published.
func (p *Postgres) MarkEnqueued(ctx context.Context, id int64) error {
	_, err := p.pool.Exec(ctx, `UPDATE orders SET job_enqueued = TRUE WHERE id = $1 AND job_enqueued = FALSE`, id)
	return err
}

// PendingEnqueue lists orders that were never enqueued.
func (p *Postgres) PendingEnqueue(ctx context.Context, olderThan time.Duration, limit int) ([]int64, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id FROM orders
		WHERE job_enqueued = FALSE AND created_at < now() - make_interval(secs => $1)
		ORDER BY created_at LIMIT $2`, olderThan.Seconds(), limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

const orderColumns = `id, customer_name, table_number, notes, status, total_cents,
	created_at, updated_at, cooking_started_at, ready_at`

func scanOrder(row pgx.Row) (order.Order, error) {
	var o order.Order
	err := row.Scan(&o.ID, &o.CustomerName, &o.TableNumber, &o.Notes, &o.Status, &o.TotalCents,
		&o.CreatedAt, &o.UpdatedAt, &o.CookingStartedAt, &o.ReadyAt)
	o.Items = []order.Item{}
	return o, err
}

// GetOrder loads one order with items.
func (p *Postgres) GetOrder(ctx context.Context, id int64) (order.Order, error) {
	o, err := scanOrder(p.pool.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return order.Order{}, order.ErrNotFound
	}
	if err != nil {
		return order.Order{}, err
	}
	orders := []order.Order{o}
	if err := p.attachItems(ctx, orders); err != nil {
		return order.Order{}, err
	}
	return orders[0], nil
}

// ListOrders lists recent orders, newest first, optionally filtered by status.
func (p *Postgres) ListOrders(ctx context.Context, f order.ListFilter) ([]order.Order, error) {
	statuses := make([]string, len(f.Statuses))
	for i, s := range f.Statuses {
		statuses[i] = string(s)
	}
	rows, err := p.pool.Query(ctx, `SELECT `+orderColumns+` FROM orders
		WHERE cardinality($1::text[]) = 0 OR status = ANY($1)
		ORDER BY created_at DESC, id DESC LIMIT $2`, statuses, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []order.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, p.attachItems(ctx, orders)
}

func (p *Postgres) attachItems(ctx context.Context, orders []order.Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]int64, len(orders))
	pos := make(map[int64]int, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
		pos[o.ID] = i
	}
	rows, err := p.pool.Query(ctx, `
		SELECT order_id, menu_item_id, item_name, unit_price_cents, quantity, line_total_cents
		FROM order_items WHERE order_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int64
		var it order.Item
		if err := rows.Scan(&orderID, &it.MenuItemID, &it.Name, &it.UnitPriceCents, &it.Quantity, &it.LineTotalCents); err != nil {
			return err
		}
		i := pos[orderID]
		orders[i].Items = append(orders[i].Items, it)
	}
	return rows.Err()
}
