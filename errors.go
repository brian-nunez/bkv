package bkv

import "errors"

var (
	ErrUnknownDriver = errors.New("UNKNOWN_DRIVER")
	ErrInvalidConfig = errors.New("INVALID_CONFIG")
	ErrKeyNotFound   = errors.New("KEY_NOT_FOUND")
	ErrStoreClosed   = errors.New("STORE_CLOSED")
)
