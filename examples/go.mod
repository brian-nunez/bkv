module bkv/testing

go 1.25.0

replace github.com/brian-nunez/bkv => ../

replace github.com/brian-nunez/bkv/drivers/local => ../drivers/local

replace github.com/brian-nunez/bkv/drivers/redis => ../drivers/redis

require (
	github.com/brian-nunez/bkv v0.0.0-00010101000000-000000000000
	github.com/brian-nunez/bkv/drivers/local v0.0.0-00010101000000-000000000000
	github.com/brian-nunez/bkv/drivers/redis v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.20.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
