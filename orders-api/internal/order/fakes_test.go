package order

import (
	"context"
	"errors"
	"time"
)

type fakeStore struct {
	menu      map[int]MenuItem
	orders    map[int64]Order
	enqueued  map[int64]bool
	nextID    int64
	createErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		menu: map[int]MenuItem{
			1: {ID: 1, Name: "Biryani", PriceCents: 60000, IsAvailable: true},
			2: {ID: 2, Name: "Chai", PriceCents: 10000, IsAvailable: true},
		},
		orders: map[int64]Order{}, enqueued: map[int64]bool{}, nextID: 100,
	}
}

func (f *fakeStore) MenuItems(_ context.Context, ids []int) (map[int]MenuItem, error) {
	out := map[int]MenuItem{}
	for _, id := range ids {
		if m, ok := f.menu[id]; ok {
			out[id] = m
		}
	}
	return out, nil
}

func (f *fakeStore) CreateOrder(_ context.Context, o NewOrder) (Order, error) {
	if f.createErr != nil {
		return Order{}, f.createErr
	}
	f.nextID++
	created := Order{ID: f.nextID, CustomerName: o.CustomerName, Status: StatusNew, TotalCents: o.TotalCents, Items: o.Items}
	f.orders[created.ID] = created
	return created, nil
}

func (f *fakeStore) MarkEnqueued(_ context.Context, id int64) error {
	f.enqueued[id] = true
	return nil
}

func (f *fakeStore) GetOrder(_ context.Context, id int64) (Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

func (f *fakeStore) ListOrders(_ context.Context, _ ListFilter) ([]Order, error) {
	out := []Order{}
	for _, o := range f.orders {
		out = append(out, o)
	}
	return out, nil
}

func (f *fakeStore) PendingEnqueue(_ context.Context, _ time.Duration, _ int) ([]int64, error) {
	var ids []int64
	for id := range f.orders {
		if !f.enqueued[id] {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (f *fakeStore) Ping(context.Context) error { return nil }

type fakeQueue struct {
	jobs []Job
	fail bool
}

func (q *fakeQueue) Enqueue(_ context.Context, j Job) error {
	if q.fail {
		return errors.New("redis down")
	}
	q.jobs = append(q.jobs, j)
	return nil
}

func (q *fakeQueue) Ping(context.Context) error { return nil }
