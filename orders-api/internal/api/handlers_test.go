package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dhaba-orders/orders-api/internal/order"
)

type memStore struct {
	orders map[int64]order.Order
	next   int64
}

func (m *memStore) MenuItems(_ context.Context, ids []int) (map[int]order.MenuItem, error) {
	all := map[int]order.MenuItem{
		1: {ID: 1, Name: "Chicken Karahi", PriceCents: 145000, IsAvailable: true},
		2: {ID: 2, Name: "Garlic Naan", PriceCents: 12000, IsAvailable: true},
	}
	out := map[int]order.MenuItem{}
	for _, id := range ids {
		if v, ok := all[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (m *memStore) CreateOrder(_ context.Context, o order.NewOrder) (order.Order, error) {
	m.next++
	created := order.Order{ID: m.next, CustomerName: o.CustomerName, TableNumber: o.TableNumber,
		Status: order.StatusNew, TotalCents: o.TotalCents, Items: o.Items, CreatedAt: time.Now()}
	m.orders[created.ID] = created
	return created, nil
}
func (m *memStore) MarkEnqueued(context.Context, int64) error { return nil }
func (m *memStore) GetOrder(_ context.Context, id int64) (order.Order, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return order.Order{}, order.ErrNotFound
}
func (m *memStore) ListOrders(_ context.Context, f order.ListFilter) ([]order.Order, error) {
	out := []order.Order{}
	for _, o := range m.orders {
		out = append(out, o)
	}
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}
func (m *memStore) PendingEnqueue(context.Context, time.Duration, int) ([]int64, error) {
	return nil, nil
}
func (m *memStore) Ping(context.Context) error { return nil }

type memQueue struct{ jobs []order.Job }

func (q *memQueue) Enqueue(_ context.Context, j order.Job) error {
	q.jobs = append(q.jobs, j)
	return nil
}
func (q *memQueue) Ping(context.Context) error { return nil }

type badPinger struct{}

func (badPinger) Ping(context.Context) error { return errors.New("connection refused") }

func setup(checks map[string]Pinger) (http.Handler, *memStore, *memQueue) {
	st, q := &memStore{orders: map[int64]order.Order{}}, &memQueue{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := order.NewService(st, q, log)
	if checks == nil {
		checks = map[string]Pinger{"postgres": st, "redis": q}
	}
	return NewServer(svc, checks, "test", []string{"http://localhost:5173"}, log).Routes(), st, q
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return m
}

func TestCreateOrder(t *testing.T) {
	h, _, q := setup(nil)
	rec := do(h, http.MethodPost, "/api/orders", `{"customerName":"Ali","tableNumber":3,"items":[{"menuItemId":1,"quantity":1},{"menuItemId":2,"quantity":2}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
	data := decode(t, rec)["data"].(map[string]any)
	if data["totalCents"].(float64) != 169000 || data["status"] != "NEW" {
		t.Errorf("unexpected body %v", data)
	}
	if rec.Header().Get("Location") != "/api/orders/1" {
		t.Errorf("Location = %q", rec.Header().Get("Location"))
	}
	if len(q.jobs) != 1 {
		t.Errorf("expected one job, got %d", len(q.jobs))
	}
}

func TestCreateOrderRejectsBadInput(t *testing.T) {
	h, _, _ := setup(nil)
	cases := map[string]struct {
		body string
		code int
		err  string
	}{
		"empty":           {"", 400, "invalid_json"},
		"malformed":       {`{"customerName":`, 400, "invalid_json"},
		"unknown field":   {`{"customerName":"A","items":[{"menuItemId":1,"quantity":1}],"totalCents":1}`, 400, "invalid_json"},
		"validation":      {`{"customerName":"","items":[]}`, 422, "validation_error"},
		"unknown item":    {`{"customerName":"A","items":[{"menuItemId":77,"quantity":1}]}`, 422, "validation_error"},
		"trailing object": {`{"customerName":"A","items":[{"menuItemId":1,"quantity":1}]}{}`, 400, "invalid_json"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(h, http.MethodPost, "/api/orders", c.body)
			if rec.Code != c.code {
				t.Fatalf("status %d want %d: %s", rec.Code, c.code, rec.Body)
			}
			if got := decode(t, rec)["error"].(map[string]any)["code"]; got != c.err {
				t.Errorf("error code %v want %s", got, c.err)
			}
		})
	}
}

func TestCreateOrderWrongContentType(t *testing.T) {
	h, _, _ := setup(nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/orders", strings.NewReader("x"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("status %d", rec.Code)
	}
}

func TestGetAndListOrders(t *testing.T) {
	h, st, _ := setup(nil)
	st.orders[5] = order.Order{ID: 5, CustomerName: "Zara", Status: order.StatusCooking, Items: []order.Item{}}
	st.next = 5

	rec := do(h, http.MethodGet, "/api/orders/5", "")
	if rec.Code != 200 || decode(t, rec)["data"].(map[string]any)["customerName"] != "Zara" {
		t.Errorf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, http.MethodGet, "/api/orders/6", ""); rec.Code != 404 {
		t.Errorf("missing order: %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/api/orders/abc", ""); rec.Code != 400 {
		t.Errorf("bad id: %d", rec.Code)
	}
	rec = do(h, http.MethodGet, "/api/orders?status=COOKING&limit=5", "")
	if rec.Code != 200 || len(decode(t, rec)["data"].([]any)) != 1 {
		t.Errorf("list: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, http.MethodGet, "/api/orders?limit=0", ""); rec.Code != 422 {
		t.Errorf("bad limit: %d", rec.Code)
	}
}

func TestChangeStatus(t *testing.T) {
	h, st, q := setup(nil)
	st.orders[9] = order.Order{ID: 9, Status: order.StatusNew}

	rec := do(h, http.MethodPatch, "/api/orders/9/status", `{"status":"cooking"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	if len(q.jobs) != 1 || q.jobs[0].TargetStatus != order.StatusCooking {
		t.Errorf("jobs %+v", q.jobs)
	}
	if rec := do(h, http.MethodPatch, "/api/orders/9/status", `{"status":"READY"}`); rec.Code != http.StatusConflict {
		t.Errorf("skip transition: %d", rec.Code)
	}
	if rec := do(h, http.MethodPatch, "/api/orders/9/status", `{"status":"EATEN"}`); rec.Code != 422 {
		t.Errorf("invalid status: %d", rec.Code)
	}
}

func TestInfo(t *testing.T) {
	t.Setenv("POD_NAME", "orders-api-abc123")
	h, _, _ := setup(nil)
	rec := do(h, http.MethodGet, "/info", "")
	data := decode(t, rec)["data"].(map[string]any)
	if rec.Code != 200 || data["pod"] != "orders-api-abc123" || data["version"] != "test" {
		t.Errorf("info: %v", data)
	}
}

func TestHealth(t *testing.T) {
	h, _, _ := setup(nil)
	if rec := do(h, http.MethodGet, "/health", ""); rec.Code != 200 || decode(t, rec)["status"] != "ok" {
		t.Errorf("healthy: %d %s", rec.Code, rec.Body)
	}
	h, _, _ = setup(map[string]Pinger{"postgres": badPinger{}})
	rec := do(h, http.MethodGet, "/health", "")
	if rec.Code != 503 || decode(t, rec)["status"] != "degraded" {
		t.Errorf("unhealthy: %d %s", rec.Code, rec.Body)
	}
}

func TestCORSAndNotFound(t *testing.T) {
	h, _, _ := setup(nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/orders", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("preflight: %d %v", rec.Code, rec.Header())
	}
	if rec := do(h, http.MethodGet, "/nope", ""); rec.Code != 404 {
		t.Errorf("not found: %d", rec.Code)
	}
}
