package http

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	if !rl.Allow("client") {
		t.Fatalf("expected first request to pass")
	}
	if !rl.Allow("client") {
		t.Fatalf("expected second request to pass")
	}
	if rl.Allow("client") {
		t.Fatalf("expected third request to be rate limited")
	}
}
