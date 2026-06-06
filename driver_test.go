package bkv

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testConfig struct {
	name string
}

func (c testConfig) DriverName() string {
	return c.name
}

type testStore struct{}

func (s *testStore) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (s *testStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return nil
}

func (s *testStore) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (s *testStore) Delete(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (s *testStore) Clear(ctx context.Context) error {
	return nil
}

func (s *testStore) Keys(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (s *testStore) HealthCheck(ctx context.Context) error {
	return nil
}

func (s *testStore) Close() error {
	return nil
}

func TestRegister(t *testing.T) {
	originalDrivers := drivers
	drivers = make(map[string]Driver)
	defer func() {
		drivers = originalDrivers
	}()

	called := false

	Register("test", func(config any) (Store, error) {
		called = true
		return &testStore{}, nil
	})

	driver, ok := drivers["test"]
	if !ok {
		t.Fatal("expected driver to be registered")
	}

	store, err := driver(testConfig{name: "test"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if store == nil {
		t.Fatal("expected store")
	}

	if !called {
		t.Fatal("expected registered driver to be called")
	}
}

func TestNewUnknownDriver(t *testing.T) {
	originalDrivers := drivers
	drivers = make(map[string]Driver)
	defer func() {
		drivers = originalDrivers
	}()

	store, err := New(testConfig{name: "missing"})

	if !errors.Is(err, ErrUnknownDriver) {
		t.Fatalf("expected ErrUnknownDriver, got %v", err)
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestNewSuccess(t *testing.T) {
	originalDrivers := drivers
	drivers = make(map[string]Driver)
	defer func() {
		drivers = originalDrivers
	}()

	expectedStore := &testStore{}

	Register("test", func(config any) (Store, error) {
		cfg, ok := config.(testConfig)
		if !ok {
			t.Fatalf("expected testConfig, got %T", config)
		}

		if cfg.name != "test" {
			t.Fatalf("expected config name %q, got %q", "test", cfg.name)
		}

		return expectedStore, nil
	})

	store, err := New(testConfig{name: "test"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if store != expectedStore {
		t.Fatalf("expected returned store %v, got %v", expectedStore, store)
	}
}

func TestNewDriverReturnsError(t *testing.T) {
	originalDrivers := drivers
	drivers = make(map[string]Driver)
	defer func() {
		drivers = originalDrivers
	}()

	expectedErr := errors.New("driver failed")

	Register("test", func(config any) (Store, error) {
		return nil, expectedErr
	})

	store, err := New(testConfig{name: "test"})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected driver error, got %v", err)
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestDriverType(t *testing.T) {
	var driver Driver = func(_ any) (Store, error) {
		return &testStore{}, nil
	}

	store, err := driver(testConfig{name: "test"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if store == nil {
		t.Fatal("expected store")
	}
}

func TestTestConfigImplementsNamedConfig(t *testing.T) {
	var _ NamedConfig = testConfig{}
}

func TestTestStoreImplementsStore(t *testing.T) {
	var _ Store = (*testStore)(nil)
}
