package order

import "testing"

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
		ok       bool
	}{
		{StatusNew, StatusCooking, true},
		{StatusCooking, StatusReady, true},
		{StatusNew, StatusReady, false},
		{StatusReady, StatusNew, false},
		{StatusReady, StatusCooking, false},
		{StatusCooking, StatusCooking, false},
	}
	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.ok {
			t.Errorf("%s -> %s: got %v want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestParseStatus(t *testing.T) {
	if _, ok := ParseStatus("COOKING"); !ok {
		t.Fatal("COOKING should parse")
	}
	if _, ok := ParseStatus("cooking"); ok {
		t.Fatal("lowercase is not a canonical status")
	}
	if _, ok := ParseStatus("DONE"); ok {
		t.Fatal("DONE is not a status")
	}
}
