package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dhaba-orders/orders-api/internal/order"
)

const maxBodyBytes = 64 << 10

// Pinger is a dependency that can be health-checked.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server wires HTTP routes to the order service.
type Server struct {
	svc       *order.Service
	checks    map[string]Pinger
	version   string
	startedAt time.Time
	log       *slog.Logger
	origins   map[string]bool
}

// NewServer builds a Server.
func NewServer(svc *order.Service, checks map[string]Pinger, version string, corsOrigins []string, log *slog.Logger) *Server {
	origins := map[string]bool{}
	for _, o := range corsOrigins {
		origins[o] = true
	}
	return &Server{svc: svc, checks: checks, version: version, startedAt: time.Now(), log: log, origins: origins}
}

// Routes returns the HTTP handler with middleware applied.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("GET /api/orders/{id}", s.getOrder)
	mux.HandleFunc("PATCH /api/orders/{id}/status", s.changeStatus)
	mux.HandleFunc("GET /info", s.info)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "route not found", nil)
	})
	return s.recoverer(s.logger(s.cors(mux)))
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req order.CreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	created, err := s.svc.PlaceOrder(r.Context(), req)
	if err != nil {
		s.handleErr(w, err)
		return
	}
	w.Header().Set("Location", "/api/orders/"+strconv.FormatInt(created.ID, 10))
	writeData(w, http.StatusCreated, created)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, err := order.ParseListFilter(q.Get("status"), q.Get("limit"))
	if err != nil {
		s.handleErr(w, err)
		return
	}
	orders, err := s.svc.List(r.Context(), f)
	if err != nil {
		s.handleErr(w, err)
		return
	}
	writeData(w, http.StatusOK, orders)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	o, err := s.svc.Get(r.Context(), id)
	if err != nil {
		s.handleErr(w, err)
		return
	}
	writeData(w, http.StatusOK, o)
}

type statusRequest struct {
	Status string `json:"status"`
}

func (s *Server) changeStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req statusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	target, valid := order.ParseStatus(strings.ToUpper(strings.TrimSpace(req.Status)))
	if !valid {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "status must be one of NEW, COOKING, READY",
			map[string]string{"status": "invalid"})
		return
	}
	o, err := s.svc.RequestStatusChange(r.Context(), id, target)
	if err != nil {
		s.handleErr(w, err)
		return
	}
	writeData(w, http.StatusAccepted, map[string]any{
		"orderId": o.ID, "currentStatus": o.Status, "requestedStatus": target,
	})
}

func (s *Server) info(w http.ResponseWriter, _ *http.Request) {
	host, _ := os.Hostname()
	writeData(w, http.StatusOK, map[string]any{
		"service":       "orders-api",
		"version":       s.version,
		"pod":           firstNonEmpty(os.Getenv("POD_NAME"), host),
		"namespace":     os.Getenv("POD_NAMESPACE"),
		"node":          os.Getenv("NODE_NAME"),
		"podIP":         os.Getenv("POD_IP"),
		"goVersion":     runtime.Version(),
		"startedAt":     s.startedAt.UTC(),
		"uptimeSeconds": int(time.Since(s.startedAt).Seconds()),
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{}
	healthy := true
	for name, p := range s.checks {
		if err := p.Ping(ctx); err != nil {
			healthy = false
			checks[name] = "error: " + err.Error()
			continue
		}
		checks[name] = "ok"
	}
	status, code := "ok", http.StatusOK
	if !healthy {
		status, code = "degraded", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "service": "orders-api", "version": s.version, "checks": checks})
}

func (s *Server) handleErr(w http.ResponseWriter, err error) {
	var ve *order.ValidationError
	var te *order.TransitionError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", ve.Error(), ve.Fields)
	case errors.As(err, &te):
		writeError(w, http.StatusConflict, "invalid_transition", te.Error(), nil)
	case errors.Is(err, order.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "order not found", nil)
	case errors.Is(err, context.Canceled):
		writeError(w, 499, "client_closed", "request cancelled", nil)
	default:
		s.log.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		switch {
		case errors.As(err, &mbe):
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body too large", nil)
		case errors.Is(err, io.EOF):
			writeError(w, http.StatusBadRequest, "invalid_json", "request body is empty", nil)
		default:
			writeError(w, http.StatusBadRequest, "invalid_json", "malformed JSON: "+err.Error(), nil)
		}
		return false
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain a single JSON object", nil)
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_id", "order id must be a positive integer", nil)
		return 0, false
	}
	return id, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.URL.Path == "/health" && rec.status == http.StatusOK {
			return
		}
		s.log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"durationMs", time.Since(start).Milliseconds())
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.log.Error("panic", "value", v, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (s.origins["*"] || s.origins[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
