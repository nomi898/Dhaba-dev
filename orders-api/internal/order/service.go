package order

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Job types placed on the Redis queue. Must match kitchen-worker.
const (
	JobOrderCreated = "ORDER_CREATED"
	JobStatusChange = "STATUS_CHANGE"
)

// Job is the queue message contract shared with kitchen-worker.
type Job struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	OrderID      int64     `json:"orderId"`
	TargetStatus Status    `json:"targetStatus,omitempty"`
	Source       string    `json:"source"`
	Attempt      int       `json:"attempt"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Store persists orders.
type Store interface {
	MenuItems(ctx context.Context, ids []int) (map[int]MenuItem, error)
	CreateOrder(ctx context.Context, o NewOrder) (Order, error)
	MarkEnqueued(ctx context.Context, id int64) error
	GetOrder(ctx context.Context, id int64) (Order, error)
	ListOrders(ctx context.Context, f ListFilter) ([]Order, error)
	PendingEnqueue(ctx context.Context, olderThan time.Duration, limit int) ([]int64, error)
	Ping(ctx context.Context) error
}

// Queue publishes jobs for kitchen-worker.
type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Ping(ctx context.Context) error
}

// TransitionError means the requested status change is not allowed.
type TransitionError struct {
	From, To Status
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("cannot move order from %s to %s", e.From, e.To)
}

// Service implements order use cases.
type Service struct {
	store Store
	queue Queue
	log   *slog.Logger
	now   func() time.Time
}

// NewService creates a Service.
func NewService(store Store, queue Queue, log *slog.Logger) *Service {
	return &Service{store: store, queue: queue, log: log, now: time.Now}
}

// PlaceOrder validates, prices, stores and enqueues a new order.
func (s *Service) PlaceOrder(ctx context.Context, req CreateRequest) (Order, error) {
	lines, err := req.Normalize()
	if err != nil {
		return Order{}, err
	}
	ids := make([]int, len(lines))
	for i, l := range lines {
		ids[i] = l.MenuItemID
	}
	menu, err := s.store.MenuItems(ctx, ids)
	if err != nil {
		return Order{}, fmt.Errorf("load menu items: %w", err)
	}
	priced, err := Price(req, lines, menu)
	if err != nil {
		return Order{}, err
	}
	created, err := s.store.CreateOrder(ctx, priced)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	s.log.Info("order created", "orderId", created.ID, "totalCents", created.TotalCents, "lines", len(created.Items))

	// The order is committed. If enqueueing fails the reconciler retries later,
	// so the customer still gets a successful response.
	if err := s.EnqueueCreated(ctx, created.ID); err != nil {
		s.log.Warn("enqueue failed; reconciler will retry", "orderId", created.ID, "err", err)
	}
	return created, nil
}

// EnqueueCreated publishes ORDER_CREATED and flags the order as enqueued.
func (s *Service) EnqueueCreated(ctx context.Context, id int64) error {
	if err := s.queue.Enqueue(ctx, s.newJob(JobOrderCreated, id, "", "orders-api")); err != nil {
		return err
	}
	return s.store.MarkEnqueued(ctx, id)
}

// RequestStatusChange validates a kitchen status change and enqueues it for the worker.
func (s *Service) RequestStatusChange(ctx context.Context, id int64, target Status) (Order, error) {
	o, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	if !o.Status.CanTransitionTo(target) {
		return Order{}, &TransitionError{From: o.Status, To: target}
	}
	if err := s.queue.Enqueue(ctx, s.newJob(JobStatusChange, id, target, "kitchen")); err != nil {
		return Order{}, fmt.Errorf("enqueue status change: %w", err)
	}
	s.log.Info("status change requested", "orderId", id, "from", o.Status, "to", target)
	return o, nil
}

// Get returns one order.
func (s *Service) Get(ctx context.Context, id int64) (Order, error) {
	return s.store.GetOrder(ctx, id)
}

// List returns recent orders.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Order, error) {
	return s.store.ListOrders(ctx, f)
}

// Reconcile re-enqueues orders whose ORDER_CREATED job was never published.
func (s *Service) Reconcile(ctx context.Context, olderThan time.Duration) (int, error) {
	ids, err := s.store.PendingEnqueue(ctx, olderThan, 100)
	if err != nil {
		return 0, err
	}
	var done int
	var errs []error
	for _, id := range ids {
		if err := s.EnqueueCreated(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("order %d: %w", id, err))
			continue
		}
		done++
	}
	if done > 0 {
		s.log.Info("reconciled orders", "count", done)
	}
	return done, errors.Join(errs...)
}

// RunReconciler calls Reconcile periodically until ctx is cancelled.
func (s *Service) RunReconciler(ctx context.Context, every, olderThan time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := s.Reconcile(ctx, olderThan); err != nil {
				s.log.Warn("reconcile failed", "err", err)
			}
		}
	}
}

func (s *Service) newJob(typ string, orderID int64, target Status, source string) Job {
	return Job{ID: newID(), Type: typ, OrderID: orderID, TargetStatus: target, Source: source, CreatedAt: s.now().UTC()}
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
