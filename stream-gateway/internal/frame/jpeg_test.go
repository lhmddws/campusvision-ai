package frame

import (
	"testing"
)

func TestYUV420P_OddDimensions(t *testing.T) {
	// 1281x721 — both odd, would panic with integer division before the fix.
	width, height := 1281, 721
	yuvSize := width * height * 3 / 2
	yuv := make([]byte, yuvSize)
	for i := range yuv {
		yuv[i] = 128 // neutral gray
	}

	// This must not panic.
	_, err := EncodeJPEG(yuv, width, height, 80)
	if err != nil {
		t.Fatalf("EncodeJPEG with odd dimensions: %v", err)
	}
}

func TestYUV420P_EvenDimensions(t *testing.T) {
	width, height := 1280, 720
	yuvSize := width * height * 3 / 2
	yuv := make([]byte, yuvSize)
	for i := range yuv {
		yuv[i] = 128
	}

	_, err := EncodeJPEG(yuv, width, height, 80)
	if err != nil {
		t.Fatalf("EncodeJPEG with even dimensions: %v", err)
	}
}

func TestYUV420P_BufferTooSmall(t *testing.T) {
	// Buffer too small should not panic, return zero-sized image.
	yuv := make([]byte, 10)
	rgba := yuvToRGBA(yuv, 1280, 720)
	if rgba.Bounds().Dx() != 0 || rgba.Bounds().Dy() != 0 {
		t.Fatal("expected zero-sized RGBA for undersized buffer")
	}
}
