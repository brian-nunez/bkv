package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/sqlite"
)

func sqlite_example() {
	conn, err := bkv.New(sqlite.Config{
		Path:   "bkv.db",
		Prefix: "myapp:",
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
