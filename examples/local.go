package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brian-nunez/bkv"
	"github.com/brian-nunez/bkv/drivers/local"
)

func local_example() {
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

	err = conn.HealthCheck(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(val)
}
