package frame

import (
	"bytes"
	"image"
	"image/jpeg"
)

// EncodeJPEG converts a raw YUV420P frame to JPEG bytes.
// quality is the JPEG quality level (1-100).
func EncodeJPEG(yuv []byte, width, height, quality int) ([]byte, error) {
	rgba := yuvToRGBA(yuv, width, height)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// yuvToRGBA converts a YUV420P planar buffer to an RGBA image using the
// BT.601 full-swing conversion formula.
func yuvToRGBA(yuv []byte, width, height int) *image.RGBA {
	// Input validation: YUV420P requires at least width*height*3/2 bytes.
	minLen := width*height + (width/2)*((height+1)/2)*2
	if len(yuv) < minLen {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	yStride := width
	uvStride := width / 2

	yPlane := yuv[:width*height]
	// Use (height+1)/2 for safety with odd frame dimensions.
	uPlane := yuv[width*height : width*height+uvStride*((height+1)/2)]
	vPlane := yuv[width*height+uvStride*((height+1)/2) : width*height+uvStride*((height+1)/2)*2]

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			yi := y*yStride + x
			ui := (y/2)*uvStride + (x / 2)
			vi := (y/2)*uvStride + (x / 2)

			Y := int32(yPlane[yi])
			U := int32(uPlane[ui])
			V := int32(vPlane[vi])

			// BT.601 JPEG conversion (full swing).
			C := Y - 16
			D := U - 128
			E := V - 128

			r := clamp((298*C + 409*E + 128) >> 8)
			g := clamp((298*C - 100*D - 208*E + 128) >> 8)
			b := clamp((298*C + 516*D + 128) >> 8)

			idx := img.PixOffset(x, y)
			img.Pix[idx+0] = r
			img.Pix[idx+1] = g
			img.Pix[idx+2] = b
			img.Pix[idx+3] = 255
		}
	}

	return img
}

// clamp restricts an int32 to the [0, 255] range.
func clamp(v int32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
