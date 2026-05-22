package domain

import "testing"

func TestEstimatePrice_MinimumFare(t *testing.T) {
	price, dist := EstimatePrice(51.0, 71.0, 51.0, 71.0)
	if dist != 0 {
		t.Fatalf("expected 0 distance, got %v", dist)
	}
	if price != 700 {
		t.Fatalf("expected minimum 700 KZT, got %v", price)
	}
}

func TestEstimatePrice_LongTrip(t *testing.T) {
	price, dist := EstimatePrice(51.128218, 71.430420, 51.500000, 72.000000)
	if dist < 10 {
		t.Fatalf("expected ~50km, got %v", dist)
	}
	if price <= 700 {
		t.Fatalf("price should scale with distance: %v", price)
	}
}

func TestRideStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{StatusRequested, StatusDriverAssigned, true},
		{StatusRequested, StatusInProgress, false},
		{StatusDriverAssigned, StatusDriverArrived, true},
		{StatusDriverArrived, StatusInProgress, true},
		{StatusInProgress, StatusCompleted, true},
		{StatusCompleted, StatusCancelled, false},
		{StatusCancelled, StatusRequested, false},
	}
	for _, c := range cases {
		got := CanTransition(c.from, c.to)
		if got != c.ok {
			t.Errorf("%s -> %s: want %v got %v", c.from, c.to, c.ok, got)
		}
	}
}
