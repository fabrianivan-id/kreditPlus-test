package repository

import "testing"

func TestErrorsDefined(t *testing.T) {
	if ErrNotFound == nil || ErrDuplicate == nil {
		t.Fatalf("expected repository errors to be defined")
	}
	if ErrNotFound == ErrDuplicate {
		t.Fatalf("expected distinct error values")
	}
}
