package entity

import "testing"

func TestLimitRemaining(t *testing.T) {
	limit := Limit{Amount: 1000, UsedAmount: 250}
	if got := limit.Remaining(); got != 750 {
		t.Fatalf("expected remaining 750, got %d", got)
	}
}
