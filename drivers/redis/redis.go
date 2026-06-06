package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"time"

	"github.com/brian-nunez/bkv"
	goredis "github.com/redis/go-redis/v9"
)

type store struct {
	client *goredis.Client
	prefix string
}

func init() {
	bkv.Register(DriverName, New)
}

func New(config any) (bkv.Store, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, bkv.ErrInvalidConfig
	}

	if cfg.Addr == "" {
		return nil, bkv.ErrInvalidConfig
	}

	opts := &goredis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.Secure {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := goredis.NewClient(opts)

	s := &store{
		client: client,
		prefix: normalizePrefix(cfg.Prefix),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.HealthCheck(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}

	return s, nil
}

func (s *store) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, s.key(key)).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", bkv.ErrKeyNotFound
		}

		return "", err
	}

	return val, err
}

func (s *store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, s.key(key), value, ttl).Err()
}

func (s *store) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, s.key(key)).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *store) Delete(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Del(ctx, s.key(key)).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *store) Clear(ctx context.Context) error {
	keys, err := s.redisKeys(ctx)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return s.client.Del(ctx, keys...).Err()
}

func (s *store) Keys(ctx context.Context) ([]string, error) {
	keys, err := s.redisKeys(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.stripPrefix(key))
	}

	return out, nil
}

func (s *store) HealthCheck(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *store) Close() error {
	return s.client.Close()
}

func (s *store) redisKeys(ctx context.Context) ([]string, error) {
	var cursor uint64
	var keys []string

	pattern := "*"
	if s.prefix != "" {
		pattern = s.prefix + "*"
	}

	for {
		batch, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, batch...)

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return keys, nil
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

var _ bkv.Store = (*store)(nil)
