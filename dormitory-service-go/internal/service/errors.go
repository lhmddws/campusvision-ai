package service

import "errors"

// Sentinel errors used across service and handler layers.
var (
	ErrNotFound           = errors.New("resource not found")
	ErrCameraLimitExceeded = errors.New("camera limit exceeded (max 50)")
	ErrInvalidRequest     = errors.New("invalid request")
)
