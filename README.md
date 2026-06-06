# bkv

`bkv` is a small driver-based key/value store abstraction for Go.

It provides a single common interface for key/value operations while allowing different storage backends to be plugged in through drivers.

Current drivers:

- `local` — in-memory map-based store
- `redis` — Redis-backed store using `github.com/redis/go-redis/v9`
- `sqlite` — SQLite-backed store using Go's `database/sql` package with a SQLite driver

## Features

- Driver registration pattern
- Redis support
- Local in-memory support
- SQLite support
- TTL support
- Key existence checks
- Delete support
- Clear all keys
- List keys
- Health checks
- Closeable stores

## Install

```bash
go get github.com/brian-nunez/bkv
```

Install the drivers you want to use:

```bash
go get github.com/brian-nunez/bkv/drivers/local
go get github.com/brian-nunez/bkv/drivers/redis
go get github.com/brian-nunez/bkv/drivers/sqlite
```

## Usage

### Redis

Import the root package and the Redis driver.

The Redis driver registers itself when imported.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/redis"
)

func main() {
	conn, err := bkv.New(redis.Config{
		Secure:   false,
		Password: "testing_password",
		Addr:     "localhost:6379",
		DB:       0,
		Prefix:   "myapp:",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ctx := context.Background()

	err = conn.Set(ctx, "testing", "hello redis", time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	val, err := conn.Get(ctx, "testing")
	if err != nil {
		if errors.Is(err, bkv.ErrKeyNotFound) {
			fmt.Println("key not found")
			return
		}

		log.Fatal(err)
	}

	fmt.Println(val)
}
```

### Local

The local driver uses an in-memory map.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/local"
)

func main() {
	conn, err := bkv.New(local.Config{})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ctx := context.Background()

	err = conn.Set(ctx, "testing", "hello local", time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	val, err := conn.Get(ctx, "testing")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(val)
}
```

### SQLite

The SQLite driver stores key/value data in a SQLite database.

It can use either a file-backed database or an in-memory database.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/sqlite"
)

func main() {
	conn, err := bkv.New(sqlite.Config{
		Path:   "bkv.db",
		Prefix: "myapp:",
		Table:  "bkv",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ctx := context.Background()

	err = conn.Set(ctx, "testing", "hello sqlite", time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	val, err := conn.Get(ctx, "testing")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(val)
}
```

### SQLite In-Memory

```go
conn, err := bkv.New(sqlite.Config{
	Path: ":memory:",
})
```

If `Path` is empty, the SQLite driver defaults to `:memory:`.

## Interface

All drivers implement the same `bkv.Store` interface.

```go
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
```

## Driver Registration

Drivers register themselves with the root `bkv` package.

Example:

```go
func init() {
	bkv.Register(DriverName, New)
}
```

This allows the app to create stores through the root package:

```go
conn, err := bkv.New(redis.Config{
	Addr: "localhost:6379",
})
```

The root package does not need to know about Redis, local storage, SQLite, or any future driver directly.

## Redis Configuration

```go
type Config struct {
	Secure   bool
	Username string
	Password string
	Addr     string
	DB       int
	Prefix   string
}
```

### Redis Fields

| Field | Description |
|---|---|
| `Secure` | Enables TLS when connecting to Redis |
| `Username` | Redis username. Optional for most instances |
| `Password` | Redis password |
| `Addr` | Redis server address, for example `localhost:6379` |
| `DB` | Redis database number |
| `Prefix` | Optional key prefix for namespacing |

Example:

```go
conn, err := bkv.New(redis.Config{
	Secure:   false,
	Password: "password",
	Addr:     "localhost:6379",
	DB:       0,
	Prefix:   "myapp:",
})
```

## Local Configuration

The local driver currently does not require configuration.

```go
conn, err := bkv.New(local.Config{})
```

The local driver stores values in memory and supports TTL expiration.

Because it is in-memory, data is lost when the process exits.

## SQLite Configuration

```go
type Config struct {
	Path   string
	Prefix string
	Table  string
}
```

### SQLite Fields

| Field | Description |
|---|---|
| `Path` | SQLite database path. If empty, `:memory:` is used |
| `Prefix` | Optional key prefix for namespacing |
| `Table` | Optional SQLite table name. If empty, `bkv` is used |

Example:

```go
conn, err := bkv.New(sqlite.Config{
	Path:   "bkv.db",
	Prefix: "myapp:",
	Table:  "bkv",
})
```

The SQLite driver creates the table automatically if it does not already exist.

The default table schema is:

```sql
CREATE TABLE IF NOT EXISTS bkv (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	expires_at INTEGER NULL
);
```

## Errors

Common errors are exposed by the root package.

```go
var (
	ErrUnknownDriver = errors.New("UNKNOWN_DRIVER")
	ErrInvalidConfig = errors.New("INVALID_CONFIG")
	ErrKeyNotFound   = errors.New("KEY_NOT_FOUND")
	ErrStoreClosed   = errors.New("STORE_CLOSED")
)
```

Example:

```go
val, err := conn.Get(ctx, "missing-key")
if errors.Is(err, bkv.ErrKeyNotFound) {
	fmt.Println("key does not exist")
}
```

## Running the Examples

Start Redis and run the examples:

```bash
cd examples
make run
```

Stop Redis and remove volumes:

```bash
make clean
```

## Docker Compose Example

```yaml
services:
  redis:
    image: redis:7.4-alpine
    container_name: redis_server
    ports:
      - "6379:6379"
    command: >
      redis-server
      --requirepass testing_password
    volumes:
      - redis_data:/data

volumes:
  redis_data:
```

## Notes

The Redis driver uses `SCAN` for key discovery instead of `KEYS`.

This is safer for larger Redis databases because `KEYS` can block Redis when the keyspace grows.

The local driver is best for tests, development, and process-local caching.

The SQLite driver is best when you want local durability without running a separate server.

## Planned Drivers

Possible future drivers:

- Valkey
- Dragonfly
- BadgerDB
- BoltDB
- DynamoDB

## License

MIT
