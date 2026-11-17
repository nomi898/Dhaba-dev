package order

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Validation limits.
const (
	MaxNameLen       = 80
	MaxNotesLen      = 500
	MaxLines         = 30
	MaxQuantity      = 20
	MaxTableNumber   = 500
	DefaultListLimit = 50
	MaxListLimit     = 200
)

// CreateRequest is the POST /api/orders payload.
type CreateRequest struct {
	CustomerName string            `json:"customerName"`
	TableNumber  *int              `json:"tableNumber"`
	Notes        string            `json:"notes"`
	Items        []CreateItemInput `json:"items"`
}

// CreateItemInput is one requested line.
type CreateItemInput struct {
	MenuItemID int `json:"menuItemId"`
	Quantity   int `json:"quantity"`
}

// ValidationError lists every problem found in a request.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+e.Fields[k])
	}
	return strings.Join(parts, "; ")
}

// Normalize trims input, checks structural rules and merges duplicate menu items.
// It returns the merged quantities keyed by menu item id, in first-seen order.
func (r *CreateRequest) Normalize() ([]CreateItemInput, error) {
	errs := map[string]string{}
	r.CustomerName = strings.TrimSpace(r.CustomerName)
	r.Notes = strings.TrimSpace(r.Notes)

	switch n := utf8.RuneCountInString(r.CustomerName); {
	case n == 0:
		errs["customerName"] = "is required"
	case n > MaxNameLen:
		errs["customerName"] = fmt.Sprintf("must be at most %d characters", MaxNameLen)
	}
	if utf8.RuneCountInString(r.Notes) > MaxNotesLen {
		errs["notes"] = fmt.Sprintf("must be at most %d characters", MaxNotesLen)
	}
	if r.TableNumber != nil && (*r.TableNumber < 1 || *r.TableNumber > MaxTableNumber) {
		errs["tableNumber"] = fmt.Sprintf("must be between 1 and %d", MaxTableNumber)
	}

	var merged []CreateItemInput
	index := map[int]int{}
	switch {
	case len(r.Items) == 0:
		errs["items"] = "must contain at least one item"
	case len(r.Items) > MaxLines:
		errs["items"] = fmt.Sprintf("must contain at most %d lines", MaxLines)
	default:
		for i, it := range r.Items {
			if it.MenuItemID <= 0 {
				errs[fmt.Sprintf("items[%d].menuItemId", i)] = "must be a positive integer"
				continue
			}
			if it.Quantity < 1 || it.Quantity > MaxQuantity {
				errs[fmt.Sprintf("items[%d].quantity", i)] = fmt.Sprintf("must be between 1 and %d", MaxQuantity)
				continue
			}
			if pos, seen := index[it.MenuItemID]; seen {
				merged[pos].Quantity += it.Quantity
				if merged[pos].Quantity > MaxQuantity {
					errs[fmt.Sprintf("items[%d].quantity", i)] = fmt.Sprintf("total quantity for menu item %d exceeds %d", it.MenuItemID, MaxQuantity)
				}
				continue
			}
			index[it.MenuItemID] = len(merged)
			merged = append(merged, it)
		}
	}

	if len(errs) > 0 {
		return nil, &ValidationError{Fields: errs}
	}
	return merged, nil
}

// Price builds a NewOrder using authoritative menu prices.
func Price(req CreateRequest, lines []CreateItemInput, menu map[int]MenuItem) (NewOrder, error) {
	errs := map[string]string{}
	out := NewOrder{CustomerName: req.CustomerName, TableNumber: req.TableNumber, Notes: req.Notes}
	for _, line := range lines {
		mi, ok := menu[line.MenuItemID]
		if !ok {
			errs[fmt.Sprintf("menuItem[%d]", line.MenuItemID)] = "does not exist"
			continue
		}
		if !mi.IsAvailable {
			errs[fmt.Sprintf("menuItem[%d]", line.MenuItemID)] = mi.Name + " is currently unavailable"
			continue
		}
		total := mi.PriceCents * line.Quantity
		out.Items = append(out.Items, Item{
			MenuItemID:     mi.ID,
			Name:           mi.Name,
			UnitPriceCents: mi.PriceCents,
			Quantity:       line.Quantity,
			LineTotalCents: total,
		})
		out.TotalCents += total
	}
	if len(errs) > 0 {
		return NewOrder{}, &ValidationError{Fields: errs}
	}
	return out, nil
}

// ParseListFilter validates query parameters for GET /api/orders.
func ParseListFilter(statusParam, limitParam string) (ListFilter, error) {
	f := ListFilter{Limit: DefaultListLimit}
	errs := map[string]string{}
	if statusParam != "" {
		for _, s := range strings.Split(statusParam, ",") {
			st, ok := ParseStatus(strings.ToUpper(strings.TrimSpace(s)))
			if !ok {
				errs["status"] = "must be a comma-separated list of NEW, COOKING, READY"
				break
			}
			f.Statuses = append(f.Statuses, st)
		}
	}
	if limitParam != "" {
		n, err := strconv.Atoi(limitParam)
		if err != nil || n < 1 || n > MaxListLimit {
			errs["limit"] = fmt.Sprintf("must be an integer between 1 and %d", MaxListLimit)
		} else {
			f.Limit = n
		}
	}
	if len(errs) > 0 {
		return ListFilter{}, &ValidationError{Fields: errs}
	}
	return f, nil
}
