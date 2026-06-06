package local

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/brian-nunez/bkv"
)

func TestNewInvalidConfig(t *testing.T) {
	conn, err := New("bad config")

	if !errors.Is(err, bkv.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	if conn != nil {
		t.Fatalf("expected nil conn, got %v", conn)
	}
}

func TestNewSuccess(t *testing.T) {
	conn, err := New(Config{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if conn == nil {
		t.Fatal("expected conn")
	}
}

func TestGetContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	val, err := s.Get(ctx, "testing")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}
}

func TestGetClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	val, err := s.Get(context.Background(), "testing")
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}
}

func TestGetMissingKey(t *testing.T) {
	s := newTestStore()

	val, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, bkv.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}
}

func TestGetExpiredKey(t *testing.T) {
	s := newTestStore()

	s.data["testing"] = item{
		value:     "hello",
		expiresAt: time.Now().Add(-time.Minute),
	}

	val, err := s.Get(context.Background(), "testing")
	if !errors.Is(err, bkv.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}

	if _, ok := s.data["testing"]; ok {
		t.Fatal("expected expired key to be deleted")
	}
}

func TestGetSuccessNoTTL(t *testing.T) {
	s := newTestStore()

	s.data["testing"] = item{
		value: "hello",
	}

	val, err := s.Get(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if val != "hello" {
		t.Fatalf("expected hello, got %q", val)
	}
}

func TestGetSuccessWithFutureTTL(t *testing.T) {
	s := newTestStore()

	s.data["testing"] = item{
		value:     "hello",
		expiresAt: time.Now().Add(time.Minute),
	}

	val, err := s.Get(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if val != "hello" {
		t.Fatalf("expected hello, got %q", val)
	}
}

func TestSetContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Set(ctx, "testing", "hello", time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSetClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	err := s.Set(context.Background(), "testing", "hello", time.Minute)
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}

func TestSetSuccessNoTTL(t *testing.T) {
	s := newTestStore()

	err := s.Set(context.Background(), "testing", "hello", 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	it, ok := s.data["testing"]
	if !ok {
		t.Fatal("expected key to exist")
	}

	if it.value != "hello" {
		t.Fatalf("expected hello, got %q", it.value)
	}

	if !it.expiresAt.IsZero() {
		t.Fatalf("expected zero expiresAt, got %v", it.expiresAt)
	}
}

func TestSetSuccessWithTTL(t *testing.T) {
	s := newTestStore()

	err := s.Set(context.Background(), "testing", "hello", time.Minute)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	it, ok := s.data["testing"]
	if !ok {
		t.Fatal("expected key to exist")
	}

	if it.value != "hello" {
		t.Fatalf("expected hello, got %q", it.value)
	}

	if it.expiresAt.IsZero() {
		t.Fatal("expected expiresAt to be set")
	}

	if time.Now().After(it.expiresAt) {
		t.Fatalf("expected expiresAt to be in the future, got %v", it.expiresAt)
	}
}

func TestExistsTrue(t *testing.T) {
	s := newTestStore()

	s.data["testing"] = item{
		value: "hello",
	}

	exists, err := s.Exists(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !exists {
		t.Fatal("expected exists to be true")
	}
}

func TestExistsFalseForMissingKey(t *testing.T) {
	s := newTestStore()

	exists, err := s.Exists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if exists {
		t.Fatal("expected exists to be false")
	}
}

func TestExistsReturnsGetError(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exists, err := s.Exists(ctx, "testing")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if exists {
		t.Fatal("expected exists to be false")
	}
}

func TestDeleteContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deleted, err := s.Delete(ctx, "testing")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}
}

func TestDeleteClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	deleted, err := s.Delete(context.Background(), "testing")
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}
}

func TestDeleteExistingKey(t *testing.T) {
	s := newTestStore()

	s.data["testing"] = item{
		value: "hello",
	}

	deleted, err := s.Delete(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !deleted {
		t.Fatal("expected deleted to be true")
	}

	if _, ok := s.data["testing"]; ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestDeleteMissingKey(t *testing.T) {
	s := newTestStore()

	deleted, err := s.Delete(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}
}

func TestClearContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Clear(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestClearClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	err := s.Clear(context.Background())
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}

func TestClearSuccess(t *testing.T) {
	s := newTestStore()

	s.data["a"] = item{value: "one"}
	s.data["b"] = item{value: "two"}

	err := s.Clear(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(s.data) != 0 {
		t.Fatalf("expected empty data, got %v", s.data)
	}
}

func TestKeysContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	keys, err := s.Keys(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}
}

func TestKeysClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	keys, err := s.Keys(context.Background())
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}
}

func TestKeysSuccessDeletesExpiredKeys(t *testing.T) {
	s := newTestStore()

	s.data["active"] = item{
		value: "hello",
	}

	s.data["expired"] = item{
		value:     "bye",
		expiresAt: time.Now().Add(-time.Minute),
	}

	keys, err := s.Keys(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"active"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}

	if _, ok := s.data["expired"]; ok {
		t.Fatal("expected expired key to be deleted")
	}
}

func TestKeysSuccessEmpty(t *testing.T) {
	s := newTestStore()

	keys, err := s.Keys(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(keys) != 0 {
		t.Fatalf("expected empty keys, got %v", keys)
	}
}

func TestHealthCheckContextCanceled(t *testing.T) {
	s := newTestStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.HealthCheck(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestHealthCheckClosed(t *testing.T) {
	s := newTestStore()
	_ = s.Close()

	err := s.HealthCheck(context.Background())
	if !errors.Is(err, bkv.ErrStoreClosed) {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}

func TestHealthCheckSuccess(t *testing.T) {
	s := newTestStore()

	err := s.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestClose(t *testing.T) {
	s := newTestStore()

	err := s.Close()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !s.closed {
		t.Fatal("expected store to be closed")
	}

	if s.data != nil {
		t.Fatalf("expected data to be nil, got %v", s.data)
	}
}

func TestExpiredNoExpiration(t *testing.T) {
	got := expired(item{})

	if got {
		t.Fatal("expected item without expiration to not be expired")
	}
}

func TestExpiredFutureExpiration(t *testing.T) {
	got := expired(item{
		expiresAt: time.Now().Add(time.Minute),
	})

	if got {
		t.Fatal("expected future expiration to not be expired")
	}
}

func TestExpiredPastExpiration(t *testing.T) {
	got := expired(item{
		expiresAt: time.Now().Add(-time.Minute),
	})

	if !got {
		t.Fatal("expected past expiration to be expired")
	}
}

func TestStoreImplementsBKVStore(t *testing.T) {
	var _ bkv.Store = (*store)(nil)
}

func newTestStore() *store {
	return &store{
		data: make(map[string]item),
	}
}
