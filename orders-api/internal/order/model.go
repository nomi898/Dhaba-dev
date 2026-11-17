// Package order contains the order domain model, validation and business logic.
package order

import (
	"errors"
	"time"
)

// Status is an order lifecycle state.
type Status string

// Order lifecycle: NEW -> COOKING -> READY.
const (
	StatusNew     Status = "NEW"
	StatusCooking Status = "COOKING"
	StatusReady   Status = "READY"
)

// ParseStatus validates a status string.
func ParseStatus(s string) (Status, bool) {
	switch Status(s) {
	case StatusNew, StatusCooking, StatusReady:
		return Status(s), true
	}
	return "", false
}

// Next returns the status that follows s, if any.
func (s Status) Next() (Status, bool) {
	switch s {
	case StatusNew:
		return StatusCooking, true
	case StatusCooking:
		return StatusReady, true
	}
	return "", false
}

// CanTransitionTo reports whether target is the immediate next status.
func (s Status) CanTransitionTo(target Status) bool {
	next, ok := s.Next()
	return ok && next == target
}

// Item is one line of an order.
type Item struct {
	MenuItemID     int    `json:"menuItemId"`
	Name           string `json:"name"`
	UnitPriceCents int    `json:"unitPriceCents"`
	Quantity       int    `json:"quantity"`
	LineTotalCents int    `json:"lineTotalCents"`
}

// Order is a persisted customer order.
type Order struct {
	ID               int64      `json:"id"`
	CustomerName     string     `json:"customerName"`
	TableNumber      *int       `json:"tableNumber"`
	Notes            string     `json:"notes"`
	Status           Status     `json:"status"`
	TotalCents       int        `json:"totalCents"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CookingStartedAt *time.Time `json:"cookingStartedAt"`
	ReadyAt          *time.Time `json:"readyAt"`
	Items            []Item     `json:"items"`
}

// MenuItem is the subset of menu data needed to price an order.
type MenuItem struct {
	ID          int
	Name        string
	PriceCents  int
	IsAvailable bool
}

// NewOrder is a validated, priced order ready to persist.
type NewOrder struct {
	CustomerName string
	TableNumber  *int
	Notes        string
	TotalCents   int
	Items        []Item
}

// ListFilter narrows order listings.
type ListFilter struct {
	Statuses []Status
	Limit    int
}

// ErrNotFound is returned when an order does not exist.
var ErrNotFound = errors.New("order not found")
