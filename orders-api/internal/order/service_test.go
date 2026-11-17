package order

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestService() (*Service, *fakeStore, *fakeQueue) {
	st, q := newFakeStore(), &fakeQueue{}
	return NewService(st, q, slog.New(slog.NewTextHandler(io.Discard, nil))), st, q
}

func TestPlaceOrderStoresAndEnqueues(t *testing.T) {
	svc, st, q := newTestService()
	o, err := svc.PlaceOrder(context.Background(), CreateRequest{
		CustomerName: "Ali", Items: []CreateItemInput{{1, 1}, {2, 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalCents != 80000 || o.Status != StatusNew {
		t.Errorf("unexpected order %+v", o)
	}
	if len(q.jobs) != 1 || q.jobs[0].Type != JobOrderCreated || q.jobs[0].OrderID != o.ID || q.jobs[0].ID == "" {
		t.Errorf("unexpected jobs %+v", q.jobs)
	}
	if !st.enqueued[o.ID] {
		t.Error("order not marked as enqueued")
	}
}

func TestPlaceOrderSucceedsWhenQueueDownAndReconcilerRecovers(t *testing.T) {
	svc, st, q := newTestService()
	q.fail = true
	o, err := svc.PlaceOrder(context.Background(), CreateRequest{CustomerName: "Ali", Items: []CreateItemInput{{1, 1}}})
	if err != nil {
		t.Fatalf("order must still be accepted: %v", err)
	}
	if st.enqueued[o.ID] {
		t.Fatal("should not be marked enqueued")
	}
	q.fail = false
	n, err := svc.Reconcile(context.Background(), time.Second)
	if err != nil || n != 1 || !st.enqueued[o.ID] || len(q.jobs) != 1 {
		t.Errorf("reconcile: n=%d err=%v jobs=%d", n, err, len(q.jobs))
	}
}

func TestPlaceOrderValidationAndStoreErrors(t *testing.T) {
	svc, st, _ := newTestService()
	var ve *ValidationError
	if _, err := svc.PlaceOrder(context.Background(), CreateRequest{CustomerName: "A", Items: []CreateItemInput{{42, 1}}}); !errors.As(err, &ve) {
		t.Errorf("unknown item should be a validation error, got %v", err)
	}
	st.createErr = errors.New("db down")
	if _, err := svc.PlaceOrder(context.Background(), CreateRequest{CustomerName: "A", Items: []CreateItemInput{{1, 1}}}); err == nil || errors.As(err, &ve) {
		t.Errorf("store error should propagate as internal error, got %v", err)
	}
}

func TestRequestStatusChange(t *testing.T) {
	svc, st, q := newTestService()
	st.orders[7] = Order{ID: 7, Status: StatusNew}

	if _, err := svc.RequestStatusChange(context.Background(), 7, StatusCooking); err != nil {
		t.Fatal(err)
	}
	if len(q.jobs) != 1 || q.jobs[0].Type != JobStatusChange || q.jobs[0].TargetStatus != StatusCooking || q.jobs[0].Source != "kitchen" {
		t.Errorf("unexpected job %+v", q.jobs)
	}

	var te *TransitionError
	if _, err := svc.RequestStatusChange(context.Background(), 7, StatusReady); !errors.As(err, &te) {
		t.Errorf("NEW -> READY must be rejected, got %v", err)
	}
	if _, err := svc.RequestStatusChange(context.Background(), 404, StatusCooking); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
