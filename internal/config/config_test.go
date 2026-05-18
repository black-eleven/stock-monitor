package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("DATA_DIR")

	cfg := Load()
	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if !strings.HasSuffix(cfg.DataDir, "data") {
		t.Errorf("data dir should end with 'data', got: %s", cfg.DataDir)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "4000")
	defer os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Port != "4000" {
		t.Errorf("expected port 4000, got %s", cfg.Port)
	}
}
