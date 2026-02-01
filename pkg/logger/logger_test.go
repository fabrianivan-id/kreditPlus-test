package logger

import "testing"

func TestNewLogger(t *testing.T) {
	if New() == nil {
		t.Fatalf("expected logger to be created")
	}
}
