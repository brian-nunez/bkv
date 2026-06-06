# bkv Redis Storage Driver

`redis` is a Redis-backed storage driver for the `bkv` key-value store abstraction, utilizing the official Go Redis client library `github.com/redis/go-redis/v9`.

---

## Installation

```bash
go get github.com/brian-nunez/bkv/drivers/redis
```

## Features

- **Distributed Storage**: Shared cache state across multiple application processes.
- **Native TTL**: Utilizes Redis's native key expiration (TTL) mechanism.
- **Secure Connections**: Supports TLS/SSL connectivity using Go's `crypto/tls`.
- **Namespacing / Key Prefixing**: Optional `Prefix` parameter to isolate keys. Prefixes are automatically formatted with a trailing colon `:` (e.g., `"myapp"` -> `"myapp:"`), and stripped when listing keys.
- **Production-Safe Key Listing**: Uses Redis `SCAN` to retrieve key lists iteratively (batch size 100), avoiding the blocking behaviour of the `KEYS` command in large databases.

## Configuration

The driver uses the [redis.Config](./config.go) struct:

| Field | Type | Description |
|---|---|---|
| `Addr` | `string` | **Required.** Redis server host/address (e.g. `"localhost:6379"`). |
| `Secure` | `bool` | Enables TLS connection configuration (MinVersion: TLS 1.2). |
| `Username`| `string` | Optional authentication username. |
| `Password`| `string` | Optional authentication password. |
| `DB` | `int` | Database number to select after connecting (defaults to `0`). |
| `Prefix` | `string` | Optional namespace prefix to prepend to all keys. |

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

## Usage Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/redis"
)

func main() {
	// Initialize the Redis store
	store, err := bkv.New(redis.Config{
		Addr:     "127.0.0.1:6379",
		Password: "mysecretpassword",
		DB:       0,
		Prefix:   "session",
	})
	if err != nil {
		log.Fatalf("Failed to initialize Redis store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set a key. The key in Redis will be formatted as "session:123"
	err = store.Set(ctx, "123", "active_session_data", 10*time.Minute)
	if err != nil {
		log.Fatalf("Set error: %v", err)
	}

	// Get the value. Auto-prepends the prefix under the hood.
	val, err := store.Get(ctx, "123")
	if err != nil {
		log.Fatalf("Get error: %v", err)
	}
	fmt.Printf("Retrieved value: %s\n", val)
}
```
