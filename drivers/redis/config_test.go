package redis

import "testing"

func TestConfigDriverName(t *testing.T) {
	cfg := Config{}

	got := cfg.DriverName()
	if got != DriverName {
		t.Fatalf("expected driver name %q, got %q", DriverName, got)
	}
}

func TestDriverNameConstant(t *testing.T) {
	if DriverName != "redis" {
		t.Fatalf("expected DriverName to be %q, got %q", "redis", DriverName)
	}
}

func TestConfigFields(t *testing.T) {
	cfg := Config{
		Secure:   true,
		Username: "test-user",
		Password: "test-password",
		Addr:     "localhost:6379",
		DB:       1,
		Prefix:   "test:",
	}

	if !cfg.Secure {
		t.Fatal("expected Secure to be true")
	}

	if cfg.Username != "test-user" {
		t.Fatalf("expected Username %q, got %q", "test-user", cfg.Username)
	}

	if cfg.Password != "test-password" {
		t.Fatalf("expected Password %q, got %q", "test-password", cfg.Password)
	}

	if cfg.Addr != "localhost:6379" {
		t.Fatalf("expected Addr %q, got %q", "localhost:6379", cfg.Addr)
	}

	if cfg.DB != 1 {
		t.Fatalf("expected DB %d, got %d", 1, cfg.DB)
	}

	if cfg.Prefix != "test:" {
		t.Fatalf("expected Prefix %q, got %q", "test:", cfg.Prefix)
	}
}
