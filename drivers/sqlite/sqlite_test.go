package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestNewInvalidTable(t *testing.T) {
	conn, err := New(Config{
		Path:  ":memory:",
		Table: "bad-table-name",
	})

	if !errors.Is(err, bkv.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	if conn != nil {
		t.Fatalf("expected nil conn, got %v", conn)
	}
}

func TestNewMigrationFailure(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS bkv (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`)).
		WillReturnError(errors.New("migration failed"))

	restore := replaceOpenDB(t, db, nil)
	defer restore()

	conn, err := New(Config{
		Path: ":memory:",
	})
	if err == nil {
		t.Fatal("expected migration error")
	}

	if conn != nil {
		t.Fatalf("expected nil conn, got %v", conn)
	}

	assertExpectations(t, mock)
}

func TestNewHealthCheckFailure(t *testing.T) {
	db, mock := newMockDBWithPing(t)

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS bkv (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPing().
		WillReturnError(errors.New("ping failed"))

	restore := replaceOpenDB(t, db, nil)
	defer restore()

	conn, err := New(Config{
		Path: ":memory:",
	})
	if err == nil {
		t.Fatal("expected health check error")
	}

	if conn != nil {
		t.Fatalf("expected nil conn, got %v", conn)
	}

	assertExpectations(t, mock)
}

func TestNewOpenFailure(t *testing.T) {
	expectedErr := errors.New("open failed")

	restore := replaceOpenDB(t, nil, expectedErr)
	defer restore()

	conn, err := New(Config{
		Path: ":memory:",
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected open error, got %v", err)
	}

	if conn != nil {
		t.Fatalf("expected nil conn, got %v", conn)
	}
}

func TestGetSuccessNoExpiration(t *testing.T) {
	s, mock := newMockStore(t)

	rows := sqlmock.NewRows([]string{"value", "expires_at"}).
		AddRow("hello", nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnRows(rows)

	val, err := s.Get(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if val != "hello" {
		t.Fatalf("expected hello, got %q", val)
	}

	assertExpectations(t, mock)
}

func TestGetSuccessFutureExpiration(t *testing.T) {
	s, mock := newMockStore(t)

	rows := sqlmock.NewRows([]string{"value", "expires_at"}).
		AddRow("hello", time.Now().Add(time.Minute).UnixNano())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnRows(rows)

	val, err := s.Get(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if val != "hello" {
		t.Fatalf("expected hello, got %q", val)
	}

	assertExpectations(t, mock)
}

func TestGetMissingKey(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	val, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, bkv.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}

	assertExpectations(t, mock)
}

func TestGetQueryError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("query failed")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnError(expectedErr)

	val, err := s.Get(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected query error, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}

	assertExpectations(t, mock)
}

func TestGetExpiredKeyDeletesAndReturnsNotFound(t *testing.T) {
	s, mock := newMockStore(t)

	rows := sqlmock.NewRows([]string{"value", "expires_at"}).
		AddRow("hello", time.Now().Add(-time.Minute).UnixNano())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnRows(rows)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnResult(sqlmock.NewResult(0, 1))

	val, err := s.Get(context.Background(), "testing")
	if !errors.Is(err, bkv.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty value, got %q", val)
	}

	assertExpectations(t, mock)
}

func TestSetNoTTL(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO bkv (key, value, expires_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   value = excluded.value,
		   expires_at = excluded.expires_at`)).
		WithArgs("testing", "hello", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Set(context.Background(), "testing", "hello", 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestSetWithTTL(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO bkv (key, value, expires_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   value = excluded.value,
		   expires_at = excluded.expires_at`)).
		WithArgs("testing", "hello", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Set(context.Background(), "testing", "hello", time.Minute)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestSetError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("set failed")

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO bkv (key, value, expires_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   value = excluded.value,
		   expires_at = excluded.expires_at`)).
		WithArgs("testing", "hello", nil).
		WillReturnError(expectedErr)

	err := s.Set(context.Background(), "testing", "hello", 0)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected set error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestExistsTrue(t *testing.T) {
	s, mock := newMockStore(t)

	rows := sqlmock.NewRows([]string{"value", "expires_at"}).
		AddRow("hello", nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnRows(rows)

	exists, err := s.Exists(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !exists {
		t.Fatal("expected exists to be true")
	}

	assertExpectations(t, mock)
}

func TestExistsFalse(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	exists, err := s.Exists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if exists {
		t.Fatal("expected exists to be false")
	}

	assertExpectations(t, mock)
}

func TestExistsGetError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("get failed")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value, expires_at FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnError(expectedErr)

	exists, err := s.Exists(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected get error, got %v", err)
	}

	if exists {
		t.Fatal("expected exists to be false")
	}

	assertExpectations(t, mock)
}

func TestDeleteExistingKey(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleted, err := s.Delete(context.Background(), "testing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !deleted {
		t.Fatal("expected deleted to be true")
	}

	assertExpectations(t, mock)
}

func TestDeleteMissingKey(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key = ?`)).
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := s.Delete(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}

	assertExpectations(t, mock)
}

func TestDeleteExecError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("delete failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnError(expectedErr)

	deleted, err := s.Delete(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}

	assertExpectations(t, mock)
}

func TestDeleteRowsAffectedError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("rows affected failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key = ?`)).
		WithArgs("testing").
		WillReturnResult(sqlmock.NewErrorResult(expectedErr))

	deleted, err := s.Delete(context.Background(), "testing")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rows affected error, got %v", err)
	}

	if deleted {
		t.Fatal("expected deleted to be false")
	}

	assertExpectations(t, mock)
}

func TestClearSuccessNoPrefix(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key LIKE ?`)).
		WithArgs("%").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Clear(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClearSuccessWithPrefix(t *testing.T) {
	s, mock := newMockStore(t)
	s.prefix = "app:"

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key LIKE ?`)).
		WithArgs("app:%").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Clear(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClearError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("clear failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv WHERE key LIKE ?`)).
		WithArgs("%").
		WillReturnError(expectedErr)

	err := s.Clear(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected clear error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestKeysSuccessNoPrefix(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"key"}).
		AddRow("a").
		AddRow("b")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key FROM bkv WHERE key LIKE ? ORDER BY key ASC`)).
		WithArgs("%").
		WillReturnRows(rows)

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

func TestKeysSuccessWithPrefix(t *testing.T) {
	s, mock := newMockStore(t)
	s.prefix = "app:"

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"key"}).
		AddRow("app:a").
		AddRow("app:b")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key FROM bkv WHERE key LIKE ? ORDER BY key ASC`)).
		WithArgs("app:%").
		WillReturnRows(rows)

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

func TestKeysDeleteExpiredError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("delete expired failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(expectedErr)

	keys, err := s.Keys(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete expired error, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}

	assertExpectations(t, mock)
}

func TestKeysQueryError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("query keys failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key FROM bkv WHERE key LIKE ? ORDER BY key ASC`)).
		WithArgs("%").
		WillReturnError(expectedErr)

	keys, err := s.Keys(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected query error, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}

	assertExpectations(t, mock)
}

func TestKeysScanError(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"key", "extra"}).
		AddRow("a", "extra-value")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key FROM bkv WHERE key LIKE ? ORDER BY key ASC`)).
		WithArgs("%").
		WillReturnRows(rows)

	keys, err := s.Keys(context.Background())
	if err == nil {
		t.Fatal("expected scan error")
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}

	assertExpectations(t, mock)
}

func TestKeysRowsError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("rows failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"key"}).
		AddRow("a").
		RowError(0, expectedErr)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key FROM bkv WHERE key LIKE ? ORDER BY key ASC`)).
		WithArgs("%").
		WillReturnRows(rows)

	keys, err := s.Keys(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rows error, got %v", err)
	}

	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}

	assertExpectations(t, mock)
}

func TestHealthCheckSuccess(t *testing.T) {
	db, mock := newMockDBWithPing(t)
	s := &store{
		db:    db,
		table: defaultTable,
	}

	mock.ExpectPing().WillReturnError(nil)

	err := s.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestHealthCheckError(t *testing.T) {
	db, mock := newMockDBWithPing(t)
	s := &store{
		db:    db,
		table: defaultTable,
	}

	expectedErr := errors.New("ping failed")

	mock.ExpectPing().WillReturnError(expectedErr)

	err := s.HealthCheck(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected ping error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestNewSuccessDefaultPathAndTable(t *testing.T) {
	db, mock := newMockDBWithPingNoCleanup(t)

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS bkv (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPing().
		WillReturnError(nil)

	mock.ExpectClose()

	restore := replaceOpenDB(t, db, nil)
	defer restore()

	conn, err := New(Config{
		Prefix: " app: ",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if conn == nil {
		t.Fatal("expected conn")
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestClose(t *testing.T) {
	db, mock := newMockDBNoCleanup(t)

	mock.ExpectClose()

	s := &store{
		db:    db,
		table: defaultTable,
	}

	err := s.Close()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func newMockDBNoCleanup(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}

	return db, mock
}

func newMockDBWithPingNoCleanup(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}

	return db, mock
}

func TestMigrateSuccess(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS bkv (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := s.migrate(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestMigrateError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("migrate failed")

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS bkv (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`)).
		WillReturnError(expectedErr)

	err := s.migrate(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected migrate error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestDeleteExpiredSuccess(t *testing.T) {
	s, mock := newMockStore(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.deleteExpired(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertExpectations(t, mock)
}

func TestDeleteExpiredError(t *testing.T) {
	s, mock := newMockStore(t)

	expectedErr := errors.New("delete expired failed")

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM bkv
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(expectedErr)

	err := s.deleteExpired(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete expired error, got %v", err)
	}

	assertExpectations(t, mock)
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

func TestNormalizeTableDefault(t *testing.T) {
	got := normalizeTable("")

	if got != defaultTable {
		t.Fatalf("expected %q, got %q", defaultTable, got)
	}
}

func TestNormalizeTableTrimsWhitespace(t *testing.T) {
	got := normalizeTable("  custom_table  ")

	if got != "custom_table" {
		t.Fatalf("expected custom_table, got %q", got)
	}
}

func TestIsExpiredInvalidNull(t *testing.T) {
	got := isExpired(sql.NullInt64{
		Valid: false,
	})

	if got {
		t.Fatal("expected invalid expiration to not be expired")
	}
}

func TestIsExpiredFuture(t *testing.T) {
	got := isExpired(sql.NullInt64{
		Valid: true,
		Int64: time.Now().Add(time.Minute).UnixNano(),
	})

	if got {
		t.Fatal("expected future expiration to not be expired")
	}
}

func TestIsExpiredPast(t *testing.T) {
	got := isExpired(sql.NullInt64{
		Valid: true,
		Int64: time.Now().Add(-time.Minute).UnixNano(),
	})

	if !got {
		t.Fatal("expected past expiration to be expired")
	}
}

func TestStoreImplementsBKVStore(t *testing.T) {
	var _ bkv.Store = (*store)(nil)
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}

func newMockDBWithPing(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}

func newMockStore(t *testing.T) (*store, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newMockDB(t)

	return &store{
		db:    db,
		table: defaultTable,
	}, mock
}

func replaceOpenDB(t *testing.T, db *sql.DB, err error) func() {
	t.Helper()

	original := openDB

	openDB = func(driverName string, dataSourceName string) (*sql.DB, error) {
		if driverName != "sqlite" {
			t.Fatalf("expected driver sqlite, got %q", driverName)
		}

		if strings.TrimSpace(dataSourceName) == "" {
			t.Fatal("expected non-empty data source name")
		}

		return db, err
	}

	return func() {
		openDB = original
	}
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
