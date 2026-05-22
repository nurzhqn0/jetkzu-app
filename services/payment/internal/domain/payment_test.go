package domain

import "testing"

func TestPaymentTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{StatusPending, StatusSucceeded, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusRefunded, true},
		{StatusSucceeded, StatusRefunded, true},
		{StatusRefunded, StatusSucceeded, false},
		{StatusFailed, StatusSucceeded, false},
	}
	for _, c := range cases {
		got := CanTransition(c.from, c.to)
		if got != c.ok {
			t.Errorf("%s -> %s: want %v got %v", c.from, c.to, c.ok, got)
		}
	}
}
