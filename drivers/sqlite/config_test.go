package sqlite

import "testing"

func TestConfigDriverName(t *testing.T) {
	cfg := Config{}

	got := cfg.DriverName()
	if got != DriverName {
		t.Fatalf("expected driver name %q, got %q", DriverName, got)
	}
}

func TestDriverNameConstant(t *testing.T) {
	if DriverName != "sqlite" {
		t.Fatalf("expected DriverName to be %q, got %q", "sqlite", DriverName)
	}
}

func TestConfigFields(t *testing.T) {
	cfg := Config{
		Path:   "/tmp/bkv.db",
		Prefix: "myapp:",
		Table:  "custom_kv",
	}

	if cfg.Path != "/tmp/bkv.db" {
		t.Fatalf("expected Path %q, got %q", "/tmp/bkv.db", cfg.Path)
	}

	if cfg.Prefix != "myapp:" {
		t.Fatalf("expected Prefix %q, got %q", "myapp:", cfg.Prefix)
	}

	if cfg.Table != "custom_kv" {
		t.Fatalf("expected Table %q, got %q", "custom_kv", cfg.Table)
	}
}
