# bkv Storage Drivers

This directory contains the storage driver implementations for `bkv` (a lightweight, driver-based key/value store abstraction in Go).

Drivers are separate Go packages that implement the [bkv.Store](../bkv.go) interface and register themselves dynamically with the core [bkv](../bkv.go) package.

---

## Existing Drivers

Each driver is hosted in its own package/sub-module. Click the links below to view their configurations and implementations:

1. **[local](./local)**
   - **Type**: Thread-safe in-memory map.
   - **Use Case**: Testing, development, or local caching where persistence is not required.
   - **Configuration**: [local.Config](./local/config.go) (no parameters needed).
   - **Features**: Supports expiration TTL with a lazy-deletion strategy during reads (`Get`, `Keys`, etc.).

2. **[redis](./redis)**
   - **Type**: Redis client wrapper using `github.com/redis/go-redis/v9`.
   - **Use Case**: Distributed systems, high-concurrency production caches, or external persistence.
   - **Configuration**: [redis.Config](./redis/config.go).
   - **Features**: TLS secure connection, password/username authentication, key prefixing (namespacing), and pagination scan-based key listing (safely avoids blocking Redis `KEYS`).

3. **[sqlite](./sqlite)**
   - **Type**: SQL-backed store using `modernc.org/sqlite` (pure Go SQLite driver).
   - **Use Case**: File-based persistent storage or serverless/lightweight deployments without dedicated Redis instances.
   - **Configuration**: [sqlite.Config](./sqlite/config.go).
   - **Features**: Automatic schema migration, table name configuration, custom prefix/namespacing, connection pooling controls, and automatic cleanup of expired records during `Keys` calls.

---

## Driver Lifecycle & Registration

`bkv` drivers register themselves with the core registry in their package `init()` function:

```go
package mydriver

import "github.com/brian-nunez/bkv"

const DriverName = "mydriver"

func init() {
	bkv.Register(DriverName, New)
}
```

When an application wants to use a driver, it imports it using a blank import (so the `init()` function runs) and invokes [bkv.New](../driver.go) with the driver's config:

```go
import (
	"github.com/brian-nunez/bkv"
	_ "github.com/brian-nunez/bkv/drivers/redis" // registers the redis driver
)

func main() {
	store, err := bkv.New(redis.Config{
		Addr: "localhost:6379",
	})
	// ...
}
```

---

## How to Implement a Custom Driver

To add a new storage backend to `bkv`, follow these steps:

### 1. Implement `bkv.NamedConfig`
Define a configuration struct that contains any connection parameters (addresses, credentials, timeouts). It must implement the [bkv.NamedConfig](../driver.go) interface:

```go
type Config struct {
	ConnectionString string
	Timeout          time.Duration
}

func (Config) DriverName() string {
	return "custom-driver-name"
}
```

### 2. Implement the `bkv.Store` Interface
Create a store type that implements the [bkv.Store](../bkv.go) interface:

```go
type customStore struct {
	// connection / database client reference
}

func (s *customStore) Get(ctx context.Context, key string) (string, error) {
	// Retrieve value. Return bkv.ErrKeyNotFound if not found.
}

func (s *customStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	// Set value with TTL. 0 or negative TTL means no expiration.
}

func (s *customStore) Exists(ctx context.Context, key string) (bool, error) {
	// Check if key exists.
}

func (s *customStore) Delete(ctx context.Context, key string) (bool, error) {
	// Delete key. Return true if deleted, false if key did not exist.
}

func (s *customStore) Clear(ctx context.Context) error {
	// Remove all keys managed by this driver instance.
}

func (s *customStore) Keys(ctx context.Context) ([]string, error) {
	// Retrieve all keys currently stored, excluding expired ones.
}

func (s *customStore) HealthCheck(ctx context.Context) error {
	// Ping backend. Return error if unhealthy.
}

func (s *customStore) Close() error {
	// Release resources/connections.
}
```

### 3. Provide the Constructor and Register the Driver
Define a factory function that takes an `any` interface, asserts it to your `Config`, sets up the connection, and registers it:

```go
func New(config any) (bkv.Store, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, bkv.ErrInvalidConfig
	}

	// 1. Establish connection/initialize driver
	// 2. Perform a connection check
	// 3. Return the store instance
	return &customStore{}, nil
}

func init() {
	bkv.Register("custom-driver-name", New)
}
```

---

## Shared Error Handling

Drivers should map their internal errors to the standard errors defined in [errors.go](../errors.go):

- Return [bkv.ErrKeyNotFound](../errors.go) when attempting to fetch a key that doesn't exist or has expired.
- Return [bkv.ErrStoreClosed](../errors.go) if operations are attempted on a closed store.
- Return [bkv.ErrInvalidConfig](../errors.go) in the factory constructor if config validation fails.
