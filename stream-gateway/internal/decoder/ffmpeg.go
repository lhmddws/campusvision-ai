package decoder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync/atomic"
)

// Decoder wraps an FFmpeg subprocess that decodes an RTSP stream into raw
// YUV420P video frames delivered over a channel.
type Decoder struct {
	rtspURL string
	width   int
	height  int
	fps     int

	cmd    *exec.Cmd
	stdout io.ReadCloser

	stopCh  chan struct{}
	running atomic.Bool
}

// NewDecoder prepares a Decoder for the given RTSP URL with the specified
// output dimensions and framerate.  Start() must be called to begin decoding.
func NewDecoder(rtspURL string, width, height, fps int) *Decoder {
	return &Decoder{
		rtspURL: rtspURL,
		width:   width,
		height:  height,
		fps:     fps,
		stopCh:  make(chan struct{}),
	}
}

// Start spawns the ffmpeg subprocess and returns a channel that delivers raw
// YUV420P frames as []byte.  Each frame is exactly width*height*3/2 bytes.
// The channel is closed when the stream ends, on read error, or when Stop() is
// called.
func (d *Decoder) Start(ctx context.Context) (<-chan []byte, error) {
	frameSize := d.width * d.height * 3 / 2
	frameCh := make(chan []byte, 2)

	args := []string{
		"-rtsp_transport", "tcp",
		"-i", d.rtspURL,
		"-an", "-sn", "-dn",
		"-f", "rawvideo",
		"-pix_fmt", "yuv420p",
		"-s", fmt.Sprintf("%dx%d", d.width, d.height),
		"-r", fmt.Sprintf("%d", d.fps),
		"-",
	}

	cmd := exec.Command("ffmpeg", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("decoder stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("decoder stderr pipe: %w", err)
	}

	d.cmd = cmd
	d.stdout = stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("decoder start ffmpeg: %w", err)
	}
	d.running.Store(true)

	// Read stderr asynchronously – ffmpeg logs diagnostic info there.
	go d.readStderr(stderr)

	// Read raw frames from stdout in a dedicated goroutine.
	go d.readFrames(frameCh, frameSize)

	return frameCh, nil
}

// Stop kills the ffmpeg subprocess and releases resources.  Safe to call
// multiple times.
func (d *Decoder) Stop() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
	}

	if d.cmd != nil && d.cmd.Process != nil {
		if err := d.cmd.Process.Kill(); err != nil {
			log.Printf("[decoder] kill ffmpeg [%s]: %v", d.rtspURL, err)
		}
	}
}

// IsRunning reports whether the decoder subprocess is currently active.
func (d *Decoder) IsRunning() bool {
	return d.running.Load()
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

// readStderr consumes ffmpeg stderr output and logs lines that indicate errors.
func (d *Decoder) readStderr(r io.ReadCloser) {
	defer r.Close()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 4096)
	for scanner.Scan() {
		line := scanner.Text()
		if isErrorLine(line) {
			log.Printf("[decoder] ffmpeg [%s]: %s", d.rtspURL, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[decoder] stderr scan error [%s]: %v", d.rtspURL, err)
	}
}

// isErrorLine returns true when the line looks like an FFmpeg error.
func isErrorLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "cannot")
}

// readFrames reads raw YUV420P frames from ffmpeg stdout and delivers them to
// the provided channel.  It exits when stopCh is closed, stdout is exhausted,
// or a read error occurs.
func (d *Decoder) readFrames(frameCh chan<- []byte, frameSize int) {
	defer func() {
		d.running.Store(false)
		close(frameCh)
		// Reap the child process so it doesn't become a zombie.
		if d.cmd != nil {
			_ = d.cmd.Wait()
		}
	}()

	buf := make([]byte, frameSize)
	reader := bufio.NewReaderSize(d.stdout, frameSize*2)

	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		// io.ReadFull guarantees we get exactly frameSize bytes or an error.
		if _, err := io.ReadFull(reader, buf); err != nil {
			return
		}

		// Copy the frame so the buffer can be reused.
		frame := make([]byte, frameSize)
		copy(frame, buf)

		select {
		case frameCh <- frame:
		case <-d.stopCh:
			return
		}
	}
}
