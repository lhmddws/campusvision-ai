package camera

import (
	"context"
	"testing"
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/config"
)

// TestStream_StopTwice verifies that calling Stop() twice does not panic.
func TestStream_StopTwice(t *testing.T) {
	s := &Stream{
		camCfg:   config.CameraConfig{ID: "test-cam"},
		stopCh:   make(chan struct{}),
		startedAt: time.Now(),
	}

	// First call — should close the channel.
	s.Stop()

	// Second call — must not panic (select-guard catches it).
	s.Stop()
}

// TestStream_StopThenRun verifies Run exits immediately when stopCh is already closed.
func TestStream_StopThenRun(t *testing.T) {
	s := &Stream{
		camCfg:   config.CameraConfig{ID: "test-cam"},
		stopCh:   make(chan struct{}),
		startedAt: time.Now(),
	}

	s.Stop()

	// Run should return immediately since stopCh is already closed.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Run(ctx)
}
