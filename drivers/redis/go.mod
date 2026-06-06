module github.com/brian-nunez/bkv/drivers/redis

go 1.25.0

replace github.com/brian-nunez/bkv => ../..

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/brian-nunez/bkv v0.0.0-00010101000000-000000000000
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/redis/go-redis/v9 v9.20.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
