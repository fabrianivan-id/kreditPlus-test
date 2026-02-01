package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	if err := os.Setenv("APP_ADDR", ":9090"); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}
	defer os.Unsetenv("APP_ADDR")

	cfg := Load()
	if cfg.AppAddr != ":9090" {
		t.Fatalf("expected APP_ADDR to be loaded")
	}
}
