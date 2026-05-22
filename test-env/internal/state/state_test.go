package state_test

import (
	"sync"
	"testing"
	"time"

	"campusvision/test-env/internal/state"
)

// ── New() initial state ────────────────────────────────────────────────────────

func TestNew_InitialState(t *testing.T) {
	s := state.New()

	// Cameras match DefaultCameras
	if len(s.Cameras) != 4 {
		t.Fatalf("expected 4 cameras, got %d", len(s.Cameras))
	}
	for id, expected := range map[string]struct {
		building, label, color string
	}{
		"cam-a": {"A", "A栋入口", "#2980b9"},
		"cam-b": {"B", "B栋入口", "#27ae60"},
		"cam-c": {"C", "C栋入口", "#8e44ad"},
		"cam-d": {"D", "D栋入口", "#e67e22"},
	} {
		got, ok := s.Cameras[id]
		if !ok {
			t.Errorf("missing camera %q", id)
			continue
		}
		if got.Building != expected.building {
			t.Errorf("camera %q Building = %q, want %q", id, got.Building, expected.building)
		}
		if got.Label != expected.label {
			t.Errorf("camera %q Label = %q, want %q", id, got.Label, expected.label)
		}
		if got.Color != expected.color {
			t.Errorf("camera %q Color = %q, want %q", id, got.Color, expected.color)
		}
	}

	// Config matches DefaultConfig
	if s.Config.FPS != 5 {
		t.Errorf("FPS = %d, want 5", s.Config.FPS)
	}
	if s.Config.FrameWidth != 640 {
		t.Errorf("FrameWidth = %d, want 640", s.Config.FrameWidth)
	}
	if s.Config.FrameHeight != 360 {
		t.Errorf("FrameHeight = %d, want 360", s.Config.FrameHeight)
	}
	if !s.Config.UseFakeData {
		t.Error("UseFakeData should be true")
	}
	if len(s.Config.TestPeople) != 6 {
		t.Errorf("len(TestPeople) = %d, want 6", len(s.Config.TestPeople))
	}

	// Internal slices are initialized (not nil)
	if events := s.GetEvents(10); events == nil {
		t.Error("GetEvents returned nil, want empty slice")
	}
	if count := s.EventCount(); count != 0 {
		t.Errorf("EventCount = %d, want 0", count)
	}
}

// ── Lock / Unlock cycles ───────────────────────────────────────────────────────

func TestLockUnlock_Cycles(t *testing.T) {
	s := state.New()

	// Repeated lock/unlock cycles must not deadlock
	for i := 0; i < 100; i++ {
		s.Lock()
		// Touch a protected field while holding write lock
		s.Config.FPS = i
		s.Unlock()
	}

	if s.GetConfig().FPS != 99 {
		t.Errorf("after cycles FPS = %d, want 99", s.GetConfig().FPS)
	}
}

func TestLockUnlock_ConcurrentReadAfterWrite(t *testing.T) {
	s := state.New()

	var wg sync.WaitGroup
	started := make(chan struct{})
	done := make(chan struct{})

	// Goroutine: acquire RLock after main goroutine releases Lock
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		s.RLock()
		defer s.RUnlock()
		close(done)
	}()

	<-started

	// Hold write lock while goroutine waits for read lock
	s.Lock()
	time.Sleep(10 * time.Millisecond) // give goroutine time to block on RLock
	s.Unlock()

	// Wait for goroutine to acquire RLock
	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("goroutine blocked trying to acquire RLock after Unlock — deadlock?")
	}

	wg.Wait()
}

func TestRLock_RUnlock_Cycles(t *testing.T) {
	s := state.New()

	for i := 0; i < 100; i++ {
		s.RLock()
		_ = s.Config.FPS
		s.RUnlock()
	}

	// Must still be usable after cycles
	if got := s.GetConfig().FPS; got != 5 {
		t.Errorf("FPS = %d, want 5", got)
	}
}

// ── SSE client lifecycle ───────────────────────────────────────────────────────

func TestSSEClient_Lifecycle(t *testing.T) {
	s := state.New()

	ch := make(chan string, 1)

	// Add — must not panic
	s.AddSSEClient(ch)

	// Broadcast must reach the client
	expected := "test-event"
	s.BroadcastSSE(expected)

	select {
	case got := <-ch:
		if got != expected {
			t.Errorf("received event = %q, want %q", got, expected)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE broadcast")
	}

	// Remove — must not panic
	s.RemoveSSEClient(ch)

	// After removal, broadcasts must not reach the client
	s.BroadcastSSE("should-not-arrive")
	select {
	case msg := <-ch:
		t.Errorf("received %q after client removal", msg)
	default:
		// expected
	}
}

func TestSSEClient_MultipleClients(t *testing.T) {
	s := state.New()

	// Add 3 clients
	clients := []chan string{
		make(chan string, 1),
		make(chan string, 1),
		make(chan string, 1),
	}

	for _, ch := range clients {
		s.AddSSEClient(ch)
	}

	// Broadcast must reach all
	s.BroadcastSSE("hello")

	for i, ch := range clients {
		select {
		case got := <-ch:
			if got != "hello" {
				t.Errorf("client %d got %q, want %q", i, got, "hello")
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}

	// Remove one, remaining should still receive
	s.RemoveSSEClient(clients[1])

	s.BroadcastSSE("world")

	// Client 0 should receive
	select {
	case got := <-clients[0]:
		if got != "world" {
			t.Errorf("client 0 got %q, want %q", got, "world")
		}
	case <-time.After(time.Second):
		t.Error("client 0 did not receive broadcast after removing client 1")
	}

	// Client 1 should NOT receive (removed)
	select {
	case msg := <-clients[1]:
		t.Errorf("removed client 1 received %q", msg)
	default:
		// expected
	}

	// Client 2 should receive
	select {
	case got := <-clients[2]:
		if got != "world" {
			t.Errorf("client 2 got %q, want %q", got, "world")
		}
	case <-time.After(time.Second):
		t.Error("client 2 did not receive broadcast")
	}
}

// ── Event logging ──────────────────────────────────────────────────────────────

func TestLogEvent_RingBuffer(t *testing.T) {
	s := state.New()

	// Log events beyond the max limit
	const overfill = 350
	for i := 0; i < overfill; i++ {
		s.LogEvent("cam-a", "entry", "test event")
	}

	count := s.EventCount()
	if count > 300 {
		t.Errorf("EventCount = %d, want ≤ 300 (ring buffer max)", count)
	}

	// Most recent event should be first
	events := s.GetEvents(3)
	if len(events) != 3 {
		t.Fatalf("GetEvents(3) returned %d events, want 3", len(events))
	}
	if events[0].CameraID != "cam-a" {
		t.Errorf("most recent event CameraID = %q, want %q", events[0].CameraID, "cam-a")
	}
	if events[0].EventType != "entry" {
		t.Errorf("most recent event EventType = %q, want %q", events[0].EventType, "entry")
	}
}
