package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/brian-nunez/bkv"

	_ "modernc.org/sqlite"
)

const defaultTable = "bkv"

var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type store struct {
	db     *sql.DB
	prefix string
	table  string
}

func init() {
	bkv.Register(DriverName, New)
}

var openDB = sql.Open

func New(config any) (bkv.Store, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, bkv.ErrInvalidConfig
	}

	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		path = ":memory:"
	}

	table := normalizeTable(cfg.Table)
	if !validTableName.MatchString(table) {
		return nil, bkv.ErrInvalidConfig
	}

	db, err := openDB("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Important for :memory: SQLite.
	// Without this, multiple connections can see different in-memory databases.
	db.SetMaxOpenConns(1)

	s := &store{
		db:     db,
		prefix: normalizePrefix(cfg.Prefix),
		table:  table,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := s.HealthCheck(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *store) Get(ctx context.Context, key string) (string, error) {
	query := fmt.Sprintf(
		`SELECT value, expires_at FROM %s WHERE key = ?`,
		s.table,
	)

	var value string
	var expiresAt sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, s.key(key)).Scan(&value, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", bkv.ErrKeyNotFound
		}

		return "", err
	}

	if isExpired(expiresAt) {
		_, _ = s.Delete(ctx, key)
		return "", bkv.ErrKeyNotFound
	}

	return value, nil
}

func (s *store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	query := fmt.Sprintf(
		`INSERT INTO %s (key, value, expires_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   value = excluded.value,
		   expires_at = excluded.expires_at`,
		s.table,
	)

	var expiresAt any
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}

	_, err := s.db.ExecContext(ctx, query, s.key(key), value, expiresAt)
	return err
}

func (s *store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Get(ctx, key)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, bkv.ErrKeyNotFound) {
		return false, nil
	}

	return false, err
}

func (s *store) Delete(ctx context.Context, key string) (bool, error) {
	query := fmt.Sprintf(
		`DELETE FROM %s WHERE key = ?`,
		s.table,
	)

	result, err := s.db.ExecContext(ctx, query, s.key(key))
	if err != nil {
		return false, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *store) Clear(ctx context.Context) error {
	query := fmt.Sprintf(
		`DELETE FROM %s WHERE key LIKE ?`,
		s.table,
	)

	_, err := s.db.ExecContext(ctx, query, s.prefix+"%")
	return err
}

func (s *store) Keys(ctx context.Context) ([]string, error) {
	if err := s.deleteExpired(ctx); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT key FROM %s WHERE key LIKE ? ORDER BY key ASC`,
		s.table,
	)

	rows, err := s.db.QueryContext(ctx, query, s.prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]string, 0)

	for rows.Next() {
		var key string

		if err := rows.Scan(&key); err != nil {
			return nil, err
		}

		keys = append(keys, s.stripPrefix(key))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func (s *store) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *store) Close() error {
	return s.db.Close()
}

func (s *store) migrate(ctx context.Context) error {
	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expires_at INTEGER NULL
		)`,
		s.table,
	)

	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (s *store) deleteExpired(ctx context.Context) error {
	query := fmt.Sprintf(
		`DELETE FROM %s
		 WHERE expires_at IS NOT NULL
		   AND expires_at <= ?`,
		s.table,
	)

	_, err := s.db.ExecContext(ctx, query, time.Now().UnixNano())
	return err
}

func (s *store) key(key string) string {
	if s.prefix == "" {
		return key
	}

	return s.prefix + key
}

func (s *store) stripPrefix(key string) string {
	if s.prefix == "" {
		return key
	}

	return strings.TrimPrefix(key, s.prefix)
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, ":")

	if prefix == "" {
		return ""
	}

	return prefix + ":"
}

func normalizeTable(table string) string {
	table = strings.TrimSpace(table)

	if table == "" {
		return defaultTable
	}

	return table
}

func isExpired(expiresAt sql.NullInt64) bool {
	if !expiresAt.Valid {
		return false
	}

	return time.Now().UnixNano() >= expiresAt.Int64
}

var _ bkv.Store = (*store)(nil)
