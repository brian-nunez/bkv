# bkv SQLite Storage Driver

`sqlite` is a SQL-backed storage driver for the `bkv` key-value store abstraction, using the pure Go SQLite driver `modernc.org/sqlite` (no CGO required).

---

## Installation

```bash
go get github.com/brian-nunez/bkv/drivers/sqlite
```

## Features

- **CGO-Free**: Runs entirely in pure Go, making cross-compilation simple and straightforward.
- **Auto-Schema Creation**: Automatically runs database migrations and table creation upon initialization.
- **In-Memory & File-Backed Support**: Supports transient in-memory databases (e.g. `":memory:"`) as well as local database files (e.g. `"bkv.db"`).
- **In-Memory Connection Pinning**: Automatically restricts database connections to `MaxOpenConns = 1` for in-memory databases to ensure all connections access the same dataset.
- **Time-to-Live (TTL)**: Stores TTL values in Unix nanoseconds (`expires_at` column) and runs cleanups during operations.
- **Namespacing / Key Prefixing**: Supports an optional namespace prefix.
- **Customizable Table Name**: Allows customizing the SQLite table name (defaults to `"bkv"`).

## Configuration

The driver uses the [sqlite.Config](./config.go) struct:

| Field | Type | Description |
|---|---|---|
| `Path` | `string` | SQLite database path. E.g., `"data.db"` or `":memory:"`. If empty, defaults to `":memory:"`. |
| `Prefix` | `string` | Optional namespace prefix to prepend to all keys (e.g., `"cache:"`). |
| `Table` | `string` | Optional name of the table to store key-value pairs (defaults to `"bkv"`). Must be a valid alphanumeric identifier. |

```go
type Config struct {
	Path   string
	Prefix string
	Table  string
}
```

## Default Schema

The table schema is automatically created by the driver:

```sql
CREATE TABLE IF NOT EXISTS [table_name] (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	expires_at INTEGER NULL
);
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
	"github.com/brian-nunez/bkv/drivers/sqlite"
)

func main() {
	// Initialize SQLite store (creates local bkv.db file)
	store, err := bkv.New(sqlite.Config{
		Path:   "bkv.db",
		Table:  "cache_store",
		Prefix: "myapp",
	})
	if err != nil {
		log.Fatalf("Failed to initialize SQLite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set value with a 1-hour expiration
	err = store.Set(ctx, "hello", "world", time.Hour)
	if err != nil {
		log.Fatalf("Set error: %v", err)
	}

	// Get value
	val, err := store.Get(ctx, "hello")
	if err != nil {
		log.Fatalf("Get error: %v", err)
	}
	fmt.Printf("Retrieved value: %s\n", val)
}
```
