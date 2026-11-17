package order

import (
	"errors"
	"strings"
	"testing"
)

func intPtr(i int) *int { return &i }

func TestNormalizeValid(t *testing.T) {
	req := CreateRequest{
		CustomerName: "  Ali  ", TableNumber: intPtr(4), Notes: " extra spicy ",
		Items: []CreateItemInput{{MenuItemID: 1, Quantity: 2}, {MenuItemID: 2, Quantity: 1}, {MenuItemID: 1, Quantity: 3}},
	}
	lines, err := req.Normalize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.CustomerName != "Ali" || req.Notes != "extra spicy" {
		t.Errorf("fields not trimmed: %q %q", req.CustomerName, req.Notes)
	}
	if len(lines) != 2 || lines[0].MenuItemID != 1 || lines[0].Quantity != 5 || lines[1].MenuItemID != 2 {
		t.Errorf("duplicate lines not merged in order: %+v", lines)
	}
}

func TestNormalizeErrors(t *testing.T) {
	cases := map[string]struct {
		req   CreateRequest
		field string
	}{
		"missing name":    {CreateRequest{Items: []CreateItemInput{{1, 1}}}, "customerName"},
		"long name":       {CreateRequest{CustomerName: strings.Repeat("a", 81), Items: []CreateItemInput{{1, 1}}}, "customerName"},
		"no items":        {CreateRequest{CustomerName: "A"}, "items"},
		"bad table":       {CreateRequest{CustomerName: "A", TableNumber: intPtr(0), Items: []CreateItemInput{{1, 1}}}, "tableNumber"},
		"zero quantity":   {CreateRequest{CustomerName: "A", Items: []CreateItemInput{{1, 0}}}, "items[0].quantity"},
		"bad menu id":     {CreateRequest{CustomerName: "A", Items: []CreateItemInput{{-3, 1}}}, "items[0].menuItemId"},
		"merged too many": {CreateRequest{CustomerName: "A", Items: []CreateItemInput{{1, 15}, {1, 10}}}, "items[1].quantity"},
		"long notes":      {CreateRequest{CustomerName: "A", Notes: strings.Repeat("n", 501), Items: []CreateItemInput{{1, 1}}}, "notes"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := c.req.Normalize()
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if _, ok := ve.Fields[c.field]; !ok {
				t.Errorf("expected error on %s, got %v", c.field, ve.Fields)
			}
		})
	}
}

func TestPriceUsesServerPrices(t *testing.T) {
	menu := map[int]MenuItem{
		1: {ID: 1, Name: "Biryani", PriceCents: 60000, IsAvailable: true},
		2: {ID: 2, Name: "Chai", PriceCents: 10000, IsAvailable: true},
	}
	req := CreateRequest{CustomerName: "Sara"}
	o, err := Price(req, []CreateItemInput{{1, 2}, {2, 3}}, menu)
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalCents != 150000 {
		t.Errorf("total = %d, want 150000", o.TotalCents)
	}
	if o.Items[0].Name != "Biryani" || o.Items[0].LineTotalCents != 120000 {
		t.Errorf("bad line: %+v", o.Items[0])
	}
}

func TestPriceRejectsUnknownAndUnavailable(t *testing.T) {
	menu := map[int]MenuItem{8: {ID: 8, Name: "Nihari", PriceCents: 1, IsAvailable: false}}
	_, err := Price(CreateRequest{CustomerName: "A"}, []CreateItemInput{{8, 1}, {99, 1}}, menu)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if !strings.Contains(ve.Fields["menuItem[8]"], "unavailable") || ve.Fields["menuItem[99]"] != "does not exist" {
		t.Errorf("unexpected fields: %v", ve.Fields)
	}
}

func TestParseListFilter(t *testing.T) {
	f, err := ParseListFilter("new, cooking", "10")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Statuses) != 2 || f.Statuses[1] != StatusCooking || f.Limit != 10 {
		t.Errorf("unexpected filter %+v", f)
	}
	if f, _ := ParseListFilter("", ""); f.Limit != DefaultListLimit || f.Statuses != nil {
		t.Errorf("defaults wrong: %+v", f)
	}
	for _, bad := range [][2]string{{"DONE", ""}, {"", "0"}, {"", "500"}, {"", "10abc"}} {
		if _, err := ParseListFilter(bad[0], bad[1]); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}
