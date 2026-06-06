package local

import "testing"

func TestConfigDriverName(t *testing.T) {
	cfg := Config{}

	got := cfg.DriverName()
	if got != DriverName {
		t.Fatalf("expected driver name %q, got %q", DriverName, got)
	}
}

func TestDriverNameConstant(t *testing.T) {
	if DriverName != "local" {
		t.Fatalf("expected DriverName to be %q, got %q", "local", DriverName)
	}
}
