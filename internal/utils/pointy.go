package utils

import (
	"go.openly.dev/pointy"
	"time"
)

// GetPointer wraps pointy.Pointer for compatibility with goctls generated code
func GetPointer[T any](v T) *T {
	return pointy.Pointer(v)
}

// GetUnixMilliPointer returns nil if input is nil, otherwise returns pointer to the value
func GetUnixMilliPointer(v int64) *int64 {
	return pointy.Pointer(v)
}

// GetTimeMilliPointer converts unix milliseconds pointer to time.Time pointer
func GetTimeMilliPointer(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	t := time.UnixMilli(*v)
	return &t
}

// TimeToUnixMilli preserves optional timestamps when reading entities.
func TimeToUnixMilli(v *time.Time) *int64 {
	if v == nil {
		return nil
	}
	value := v.UnixMilli()
	return &value
}
