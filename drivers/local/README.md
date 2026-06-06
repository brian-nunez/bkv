# bkv Local Storage Driver

`local` is a thread-safe, in-memory storage driver for the `bkv` key-value store abstraction.

---

## Installation

```bash
go get github.com/brian-nunez/bkv/drivers/local
```

## Features

- **Thread-Safety**: Utilizes a `sync.RWMutex` to guard concurrent read and write operations.
- **TTL Expiration**: Supports key-level Time-To-Live (TTL) using a lazy-deletion strategy (expired keys are filtered and deleted during read operations like `Get`, `Exists`, and `Keys`).
- **No Dependencies**: Pure standard-library Go implementation with zero external dependencies.

## Configuration

The driver uses the [local.Config](./config.go) struct, which does not require any parameters.

```go
type Config struct{}
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
	"github.com/brian-nunez/bkv/drivers/local"
)

func main() {
	// Initialize the local store
	store, err := bkv.New(local.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize local store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set a value with a 5-second TTL
	err = store.Set(ctx, "session:123", "user_active", 5*time.Second)
	if err != nil {
		log.Fatalf("Set error: %v", err)
	}

	// Get the value
	val, err := store.Get(ctx, "session:123")
	if err != nil {
		log.Fatalf("Get error: %v", err)
	}
	fmt.Printf("Retrieved value: %s\n", val)
}
```
