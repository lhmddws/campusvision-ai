package camera

import (
	"context"
	"encoding/base64"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/config"
	"github.com/sims/campusvision/stream-gateway/internal/decoder"
	"github.com/sims/campusvision/stream-gateway/internal/frame"
	"github.com/sims/campusvision/stream-gateway/internal/kafka"
)

type Stream struct {
	camCfg   config.CameraConfig
	frameCfg config.FrameConfig
	rtspCfg  config.RTSPConfig
	producer *kafka.Producer

	updateStatus func(string, CameraStatus)

	stopCh     chan struct{}
	frameCount atomic.Int64
	startedAt  time.Time
	connected  atomic.Bool

	mu  sync.Mutex
	dec *decoder.Decoder
}

func NewStream(
	camCfg config.CameraConfig,
	frameCfg config.FrameConfig,
	rtspCfg config.RTSPConfig,
	producer *kafka.Producer,
	updateStatus func(string, CameraStatus),
) *Stream {
	return &Stream{
		camCfg:       camCfg,
		frameCfg:     frameCfg,
		rtspCfg:      rtspCfg,
		producer:     producer,
		updateStatus: updateStatus,
		stopCh:       make(chan struct{}),
	}
}

func (s *Stream) Run(ctx context.Context) {
	s.startedAt = time.Now()
	s.connected.Store(false)

	ext := frame.NewExtractor(s.frameCfg)
	reconnectCount := 0

	// outer loop: reconnect on stream failure
	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Check max reconnect attempts (0 = infinite).
		if s.rtspCfg.MaxReconnectAttempts > 0 && reconnectCount >= s.rtspCfg.MaxReconnectAttempts {
			log.Printf("[stream] max reconnect attempts (%d) reached for %s", s.rtspCfg.MaxReconnectAttempts, s.camCfg.ID)
			return
		}

		dec := decoder.NewDecoder(
			s.camCfg.RTSPURL,
			s.frameCfg.Width,
			s.frameCfg.Height,
			s.frameCfg.FPSDay, // decode at max rate; extractor handles rate limiting
		)

		s.mu.Lock()
		s.dec = dec
		s.mu.Unlock()

		log.Printf("[stream] starting decoder for %s (%s栋) – %s", s.camCfg.ID, s.camCfg.Building, s.camCfg.RTSPURL)
		frameCh, err := dec.Start(ctx)
		if err != nil {
			log.Printf("[stream] decoder start error [%s]: %v", s.camCfg.ID, err)
			s.connected.Store(false)
			s.emitStatusUpdate()
			reconnectCount++
			s.reconnectWait(ctx)
			continue
		}

		reconnectCount = 0 // reset on successful start
		s.connected.Store(true)

		s.processFrames(ctx, dec, frameCh, ext)
	}
}

// processFrames reads raw frames from the decoder channel, runs motion
// detection via the extractor, encodes selected frames as JPEG, and sends
// them to Kafka.
func (s *Stream) processFrames(ctx context.Context, dec *decoder.Decoder, frameCh <-chan []byte, ext *frame.Extractor) {
	var lastStatusUpdate time.Time

	for {
		select {
		case <-s.stopCh:
			dec.Stop()
			return
		case <-ctx.Done():
			dec.Stop()
			return
		case rawFrame, ok := <-frameCh:
			if !ok {
				// Channel closed – stream ended or decoder stopped.
				s.connected.Store(false)
				s.emitStatusUpdate()
				log.Printf("[stream] frame channel closed for %s, reconnecting...", s.camCfg.ID)
				return
			}

			s.connected.Store(true)

			shouldCapture, motionScore := ext.ShouldCapture(rawFrame, s.frameCfg.Width, s.frameCfg.Height)
			if !shouldCapture {
				continue
			}

			seq := s.frameCount.Add(1)

			jpegBytes, err := frame.EncodeJPEG(rawFrame, s.frameCfg.Width, s.frameCfg.Height, s.frameCfg.JPEGQuality)
			if err != nil {
				log.Printf("[stream] JPEG encode error [%s]: %v", s.camCfg.ID, err)
				continue
			}

			frameData := base64.StdEncoding.EncodeToString(jpegBytes)
			isDynamic := motionScore > s.frameCfg.MotionThreshold

			msg := kafka.FrameMessage{
				CameraID:      s.camCfg.ID,
				Building:      s.camCfg.Building,
				Timestamp:     time.Now().UnixMilli(),
				FrameSequence: seq,
				FrameData:     frameData,
				FrameWidth:    s.frameCfg.Width,
				FrameHeight:   s.frameCfg.Height,
				JPEGQuality:   s.frameCfg.JPEGQuality,
				IsDynamic:     isDynamic && s.frameCfg.DynamicExtraction,
			}

			if err := s.producer.SendFrame(ctx, msg); err != nil {
				log.Printf("[stream] send frame error [%s]: %v", s.camCfg.ID, err)
			}

			// Periodic status update – every 30 frames or at least every 5 s.
			if s.frameCount.Load()%30 == 0 || time.Since(lastStatusUpdate) > 5*time.Second {
				s.emitStatusUpdate()
				lastStatusUpdate = time.Now()
			}
		}
	}
}

// Stop signals the Run goroutine to shut down and stops the active decoder.
func (s *Stream) Stop() {
	close(s.stopCh)

	s.mu.Lock()
	if s.dec != nil {
		s.dec.Stop()
		s.dec = nil
	}
	s.mu.Unlock()
}

// IsConnected reports whether the stream is currently receiving frames.
func (s *Stream) IsConnected() bool {
	return s.connected.Load()
}

// FramesSent returns the total number of frames successfully sent to Kafka.
func (s *Stream) FramesSent() int64 {
	return s.frameCount.Load()
}

// Uptime returns the duration since Run() was called.
func (s *Stream) Uptime() time.Duration {
	return time.Since(s.startedAt)
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

func (s *Stream) emitStatusUpdate() {
	if s.updateStatus == nil {
		return
	}
	uptime := time.Since(s.startedAt).Seconds()
	fps := float64(s.frameCount.Load()) / uptime
	s.updateStatus(s.camCfg.ID, CameraStatus{
		CameraID:      s.camCfg.ID,
		Building:      s.camCfg.Building,
		Connected:     s.connected.Load(),
		FPS:           fps,
		LastFrameTime: time.Now().Format(time.RFC3339),
		FramesSent:    s.frameCount.Load(),
		UptimeSeconds: int64(uptime),
	})
}

func (s *Stream) reconnectWait(ctx context.Context) {
	timer := time.NewTimer(s.rtspCfg.ReconnectInterval)
	defer timer.Stop()

	select {
	case <-s.stopCh:
	case <-ctx.Done():
	case <-timer.C:
	}
}
