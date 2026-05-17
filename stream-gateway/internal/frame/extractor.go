package frame

import (
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/config"
)

// Extractor implements dynamic frame extraction with motion detection based on
// luminance (Y-plane) frame differencing.
type Extractor struct {
	motionThreshold float64
	prevFrame       []byte // downsampled luminance from previous call
	frameCount      uint64

	lastCaptureTime time.Time
	fpsDay          int
	fpsNight        int
	dynamicEnabled  bool
}

// NewExtractor creates an Extractor from the provided FrameConfig.
func NewExtractor(cfg config.FrameConfig) *Extractor {
	return &Extractor{
		motionThreshold: cfg.MotionThreshold,
		fpsDay:          cfg.FPSDay,
		fpsNight:        cfg.FPSNight,
		dynamicEnabled:  cfg.DynamicExtraction,
		lastCaptureTime: time.Now(), // avoid immediate burst
	}
}

// ShouldCapture decides whether the given YUV420P frame should be captured
// (and thus encoded / sent) based on motion and rate-limiting rules.
//
// It returns (shouldCapture bool, motionScore float64) where motionScore is
// the mean absolute luminance difference ratio (0.0–1.0) between the current
// and previous downsampled frame.
//
// Rules:
//   - prevFrame is updated on every call regardless of the capture decision.
//   - When dynamic extraction is enabled, frames are only captured when motion
//     exceeds the threshold, subject to the FPS limit for the current time of
//     day.
//   - When dynamic extraction is disabled, frames are captured at a steady
//     rate determined by GetCurrentFPS().
func (e *Extractor) ShouldCapture(yuv []byte, width, height int) (bool, float64) {
	// Extract and downsample the luminance (Y) plane.
	yPlane := yuv[:width*height]
	ds := downsampleY(yPlane, width, height, 160, 90)

	// Compute motion score against the previous downsampled frame.
	motionScore := 0.0
	if e.prevFrame != nil {
		motionScore = meanAbsDiff(ds, e.prevFrame)
	}
	e.prevFrame = ds

	// Rate-limiting: determine the minimum interval between captures.
	currentFPS := e.GetCurrentFPS()
	var minInterval time.Duration
	if currentFPS > 0 {
		minInterval = time.Second / time.Duration(currentFPS)
	} else {
		minInterval = 0
	}

	now := time.Now()
	elapsed := now.Sub(e.lastCaptureTime)

	if e.dynamicEnabled {
		// Only capture on motion, but never exceed the FPS limit.
		if motionScore > e.motionThreshold && elapsed >= minInterval {
			e.lastCaptureTime = now
			e.frameCount++
			return true, motionScore
		}
		return false, motionScore
	}

	// Static extraction: capture at a fixed FPS regardless of motion.
	if elapsed >= minInterval {
		e.lastCaptureTime = now
		e.frameCount++
		return true, motionScore
	}
	return false, motionScore
}

// GetCurrentFPS returns the target frame rate based on the time of day:
//   - Day (06:00–21:59): fpsDay
//   - Night (22:00–05:59): fpsNight
func (e *Extractor) GetCurrentFPS() int {
	h := time.Now().Hour()
	if h >= 22 || h < 6 {
		return e.fpsNight
	}
	return e.fpsDay
}

// ---------------------------------------------------------------------------
// motion detection helpers
// ---------------------------------------------------------------------------

// downsampleY resamples the luminance plane (origW×origH) to a smaller size
// (targetW×targetH) using nearest-neighbour interpolation.
func downsampleY(yPlane []byte, origW, origH, targetW, targetH int) []byte {
	out := make([]byte, targetW*targetH)
	xRatio := float64(origW) / float64(targetW)
	yRatio := float64(origH) / float64(targetH)

	for ty := 0; ty < targetH; ty++ {
		for tx := 0; tx < targetW; tx++ {
			ox := int(float64(tx) * xRatio)
			oy := int(float64(ty) * yRatio)
			if ox >= origW {
				ox = origW - 1
			}
			if oy >= origH {
				oy = origH - 1
			}
			out[ty*targetW+tx] = yPlane[oy*origW+ox]
		}
	}
	return out
}

// meanAbsDiff returns the mean absolute difference between two equal-length
// byte slices, normalised to the [0, 1] range (dividing by 255).
func meanAbsDiff(a, b []byte) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum int
	for i := range a {
		diff := int(a[i]) - int(b[i])
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}
	return float64(sum) / float64(len(a)) / 255.0
}
