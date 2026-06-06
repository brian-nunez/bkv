package redis

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/brian-nunez/bkv"
	"github.com/go-redis/redismock/v9"
	goredis "github.com/redis/go-redis/v9"
)

func TestNewInvalidConfigType(t *testing.T) {
	store, err := New("bad config")

	if !errors.Is(err, bkv.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestNewInvalidEmptyAddr(t *testing.T) {
	store, err := New(Config{})

	if !errors.Is(err, bkv.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestNewHealthCheckFailure(t *testing.T) {
	addr := closedLocalAddr(t)

	store, err := New(Config{
		Addr: addr,
	})

	if err == nil {
		t.Fatal("expected health check error")
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestNewSecureHealthCheckFailure(t *testing.T) {
	addr := closedLocalAddr(t)

	store, err := New(Config{
		Addr:   addr,
		Secure: true,
	})

	if err == nil {
		t.Fatal("expected health check error")
	}

	if store != nil {
		t.Fatalf("expected nil store, got %v", store)
	}
}

func TestNewSuccess(t *testing.T) {
	server := miniredis.RunT(t)

	store, err := New(Config{
		Addr:   server.Addr(),
		Prefix: " test-prefix: ",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if store == nil {
		t.Fatal("expected store")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}
}

func TestGetSuccess(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectGet("app:testing").SetVal("hello")

	got, err := s.Get(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}

	assertExpectations(t, mock)
}

func TestGetKeyNotFound(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectGet("missing").RedisNil()

	got, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, bkv.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	if got != "" {
		t.Fatalf("expected empty value, got %q", got)
	}

	assertExpectations(t, mock)
}

func TestGetRedisError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("redis exploded")

	mock.ExpectGet("testing").SetErr(expectedErr)

	got, err := s.Get(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected redis error, got %v", err)
	}

	if got != "" {
		t.Fatalf("expected empty value, got %q", got)
	}

	assertExpectations(t, mock)
}

func TestSetSuccess(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectSet("app:testing", "hello", time.Minute).SetVal("OK")

	err := s.Set(context.Background(), "testing", "hello", time.Minute)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestSetError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("set failed")

	mock.ExpectSet("testing", "hello", time.Minute).SetErr(expectedErr)

	err := s.Set(context.Background(), "testing", "hello", time.Minute)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected set error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestExistsTrue(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectExists("app:testing").SetVal(1)

	exists, err := s.Exists(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !exists {
		t.Fatal("expected key to exist")
	}

	assertExpectations(t, mock)
}

func TestExistsFalse(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectExists("testing").SetVal(0)

	exists, err := s.Exists(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if exists {
		t.Fatal("expected key to not exist")
	}

	assertExpectations(t, mock)
}

func TestExistsError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("exists failed")

	mock.ExpectExists("testing").SetErr(expectedErr)

	exists, err := s.Exists(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected exists error, got %v", err)
	}

	if exists {
		t.Fatal("expected exists to be false")
	}

	assertExpectations(t, mock)
}

func TestDeleteTrue(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectDel("app:testing").SetVal(1)

	deleted, err := s.Delete(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !deleted {
		t.Fatal("expected deleted to be true")
	}

	assertExpectations(t, mock)
}

func TestDeleteFalse(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectDel("testing").SetVal(0)

	deleted, err := s.Delete(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}

	assertExpectations(t, mock)
}

func TestDeleteError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("delete failed")

	mock.ExpectDel("testing").SetErr(expectedErr)

	deleted, err := s.Delete(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}

	assertExpectations(t, mock)
}

func TestClearNoKeys(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectScan(0, "*", 100).SetVal([]string{}, 0)

	err := s.Clear(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClearWithKeys(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectScan(0, "app:*", 100).SetVal([]string{"app:a", "app:b"}, 0)
	mock.ExpectDel("app:a", "app:b").SetVal(2)

	err := s.Clear(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClearScanError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("scan failed")

	mock.ExpectScan(0, "*", 100).SetErr(expectedErr)

	err := s.Clear(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected scan error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClearDeleteError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("del failed")

	mock.ExpectScan(0, "*", 100).SetVal([]string{"a", "b"}, 0)
	mock.ExpectDel("a", "b").SetErr(expectedErr)

	err := s.Clear(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestKeysNoPrefix(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectScan(0, "*", 100).SetVal([]string{"a", "b"}, 0)

	keys, err := s.Keys(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"a", "b"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}

	assertExpectations(t, mock)
}

func TestKeysWithPrefix(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectScan(0, "app:*", 100).SetVal([]string{"app:a", "app:b"}, 0)

	keys, err := s.Keys(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"a", "b"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}

	assertExpectations(t, mock)
}

func TestKeysScanError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("scan failed")

	mock.ExpectScan(0, "*", 100).SetErr(expectedErr)

	keys, err := s.Keys(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected scan error, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}

	assertExpectations(t, mock)
}

func TestRedisKeysMultipleScans(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
		prefix: "app:",
	}

	mock.ExpectScan(0, "app:*", 100).SetVal([]string{"app:a"}, 42)
	mock.ExpectScan(42, "app:*", 100).SetVal([]string{"app:b"}, 0)

	keys, err := s.redisKeys(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	expected := []string{"app:a", "app:b"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}

	assertExpectations(t, mock)
}

func TestHealthCheckSuccess(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	mock.ExpectPing().SetVal("PONG")

	err := s.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestHealthCheckError(t *testing.T) {
	client, mock := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	expectedErr := errors.New("ping failed")

	mock.ExpectPing().SetErr(expectedErr)

	err := s.HealthCheck(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected ping error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClose(t *testing.T) {
	client, _ := redismock.NewClientMock()

	s := &store{
		client: client,
	}

	err := s.Close()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestKeyNoPrefix(t *testing.T) {
	s := &store{}

	got := s.key("testing")
	if got != "testing" {
		t.Fatalf("expected testing, got %q", got)
	}
}

func TestKeyWithPrefix(t *testing.T) {
	s := &store{
		prefix: "app:",
	}

	got := s.key("testing")
	if got != "app:testing" {
		t.Fatalf("expected app:testing, got %q", got)
	}
}

func TestStripPrefixNoPrefix(t *testing.T) {
	s := &store{}

	got := s.stripPrefix("testing")
	if got != "testing" {
		t.Fatalf("expected testing, got %q", got)
	}
}

func TestStripPrefixWithPrefix(t *testing.T) {
	s := &store{
		prefix: "app:",
	}

	got := s.stripPrefix("app:testing")
	if got != "testing" {
		t.Fatalf("expected testing, got %q", got)
	}
}

func TestNormalizePrefixEmpty(t *testing.T) {
	got := normalizePrefix("")
	if got != "" {
		t.Fatalf("expected empty prefix, got %q", got)
	}
}

func TestNormalizePrefixWhitespace(t *testing.T) {
	got := normalizePrefix("   ")
	if got != "" {
		t.Fatalf("expected empty prefix, got %q", got)
	}
}

func TestNormalizePrefixTrimsColonsAndAddsOneColon(t *testing.T) {
	got := normalizePrefix("::app::")
	if got != "app:" {
		t.Fatalf("expected app:, got %q", got)
	}
}

func TestNormalizePrefixTrimsWhitespaceAndColons(t *testing.T) {
	got := normalizePrefix("  :app:  ")
	if got != "app:" {
		t.Fatalf("expected app:, got %q", got)
	}
}

func TestStoreImplementsBKVStore(t *testing.T) {
	var _ bkv.Store = (*store)(nil)
}

func TestUsesGoRedisNilMapping(t *testing.T) {
	if !errors.Is(goredis.Nil, goredis.Nil) {
		t.Fatal("expected go redis nil to match itself")
	}
}

func assertExpectations(t *testing.T, mock redismock.ClientMock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis mock expectations: %v", err)
	}
}

func closedLocalAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	addr := listener.Addr().String()

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	return addr
}
