package local

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/brian-nunez/bkv"
)

type store struct {
	mu     sync.RWMutex
	data   map[string]item
	closed bool
}

type item struct {
	value     string
	expiresAt time.Time
}

func init() {
	bkv.Register(DriverName, New)
}

func New(config any) (bkv.Store, error) {
	_, ok := config.(Config)
	if !ok {
		return nil, bkv.ErrInvalidConfig
	}

	return &store{
		data: make(map[string]item),
	}, nil
}

func (s *store) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return "", bkv.ErrStoreClosed
	}

	it, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return "", bkv.ErrKeyNotFound
	}

	if expired(it) {
		_, _ = s.Delete(ctx, key)
		return "", bkv.ErrKeyNotFound
	}

	return it.value, nil
}

func (s *store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return bkv.ErrStoreClosed
	}

	s.data[key] = item{
		value:     value,
		expiresAt: expiresAt,
	}

	return nil
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
	if err := ctx.Err(); err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false, bkv.ErrStoreClosed
	}

	_, ok := s.data[key]
	delete(s.data, key)

	return ok, nil
}

func (s *store) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return bkv.ErrStoreClosed
	}

	s.data = make(map[string]item)

	return nil
}

func (s *store) Keys(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, bkv.ErrStoreClosed
	}

	keys := make([]string, 0, len(s.data))

	for key, it := range s.data {
		if expired(it) {
			delete(s.data, key)
			continue
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (s *store) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return bkv.ErrStoreClosed
	}

	return nil
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.data = nil

	return nil
}

func expired(it item) bool {
	return !it.expiresAt.IsZero() && time.Now().After(it.expiresAt)
}

var _ bkv.Store = (*store)(nil)
