module bkv/testing

go 1.25.0

// use for local testing
// replace github.com/brian-nunez/bkv => ../
//
// replace github.com/brian-nunez/bkv/drivers/local => ../drivers/local
//
// replace github.com/brian-nunez/bkv/drivers/redis => ../drivers/redis

// use for testing tagged versions
require (
	github.com/brian-nunez/bkv v1.0.2
	github.com/brian-nunez/bkv/drivers/local v1.0.2
	github.com/brian-nunez/bkv/drivers/redis v1.0.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.20.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
