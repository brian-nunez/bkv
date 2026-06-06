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

func redis_example() {
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
