package http

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	body := strings.NewReader(`{"known": "value", "unknown": 1}`)
	req := httptest.NewRequest("POST", "/", body)

	var payload struct {
		Known string `json:"known"`
	}

	err := decodeJSON(req, &payload)
	if err == nil {
		t.Fatalf("expected error for unknown field")
	}
}

func TestWriteJSONSetsStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, 200, map[string]string{"status": "ok"})

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType == "" {
		t.Fatalf("expected Content-Type to be set")
	}

	if !bytes.Contains(rec.Body.Bytes(), []byte("\"status\"")) {
		t.Fatalf("expected JSON body to contain status")
	}
}
