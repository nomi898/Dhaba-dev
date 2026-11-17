// Command server runs the orders-api HTTP service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dhaba-orders/orders-api/internal/api"
	"github.com/dhaba-orders/orders-api/internal/config"
	"github.com/dhaba-orders/orders-api/internal/order"
	"github.com/dhaba-orders/orders-api/internal/queue"
	"github.com/dhaba-orders/orders-api/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := store.New(ctx, cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB,
		DialTimeout: 3 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second,
	})
	defer func() { _ = rdb.Close() }()
	q := queue.New(rdb)

	svc := order.NewService(pg, q, log)
	go svc.RunReconciler(ctx, cfg.ReconcileInterval, cfg.ReconcileAfter)

	srv := api.NewServer(svc, map[string]api.Pinger{"postgres": pg, "redis": q}, cfg.AppVersion, cfg.CORSOrigins, log)
	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("orders-api listening", "port", cfg.Port, "version", cfg.AppVersion)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}
