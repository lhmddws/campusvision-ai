package service

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors used across service and handler layers.
var (
	ErrNotFound            = errors.New("resource not found")
	ErrCameraLimitExceeded = errors.New("camera limit exceeded (max 50)")
	ErrInvalidRequest      = errors.New("invalid request")
)

func serviceCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() { <-ctx.Done(); cancel() }()
	return ctx
}
