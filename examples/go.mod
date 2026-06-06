module bkv/testing

go 1.25.0

// use for local testing
replace github.com/brian-nunez/bkv => ../

replace github.com/brian-nunez/bkv/drivers/local => ../drivers/local

replace github.com/brian-nunez/bkv/drivers/redis => ../drivers/redis

replace github.com/brian-nunez/bkv/drivers/sqlite => ../drivers/sqlite

require (
	github.com/brian-nunez/bkv v1.0.2
	github.com/brian-nunez/bkv/drivers/local v0.0.0-00010101000000-000000000000
	github.com/brian-nunez/bkv/drivers/redis v0.0.0-00010101000000-000000000000
	github.com/brian-nunez/bkv/drivers/sqlite v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/redis/go-redis/v9 v9.20.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.51.0 // indirect
)
