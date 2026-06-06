package bkv

import (
	"context"
	"time"
)

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) (deleted bool, err error)
	Clear(ctx context.Context) error
	Keys(ctx context.Context) ([]string, error)
	HealthCheck(ctx context.Context) error
	Close() error
}
