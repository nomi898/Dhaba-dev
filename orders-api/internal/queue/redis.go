// Package queue publishes order jobs to Redis for kitchen-worker.
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/dhaba-orders/orders-api/internal/order"
)

// Key is the pending-jobs list. kitchen-worker consumes from the right (FIFO).
const Key = "orders:queue"

// Redis implements order.Queue.
type Redis struct {
	client *redis.Client
}

// New wraps a go-redis client.
func New(client *redis.Client) *Redis { return &Redis{client: client} }

// Enqueue pushes a job onto the queue.
func (r *Redis) Enqueue(ctx context.Context, job order.Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return r.client.LPush(ctx, Key, payload).Err()
}

// Ping checks connectivity.
func (r *Redis) Ping(ctx context.Context) error { return r.client.Ping(ctx).Err() }
