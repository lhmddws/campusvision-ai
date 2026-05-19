package simulation

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"campusvision/test-env/internal/state"
)

// drawText draws a string on an RGBA image using a built-in bitmap font.
func drawText(img *image.RGBA, x, y int, text string, col color.RGBA) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)
}

// GenerateFrame creates a simulated camera frame as JPEG bytes.
func GenerateFrame(st *state.State, cameraID string, person string, action string) ([]byte, error) {
	st.RLock()
	cfg := st.Config
	cam, ok := st.Cameras[cameraID]
	if !ok {
		cam = state.CameraDef{Building: "?", Label: "Unknown", Color: "#555577"}
	}
	people := make([]string, len(cfg.TestPeople))
	copy(people, cfg.TestPeople)
	st.RUnlock()

	W := cfg.FrameWidth
	H := cfg.FrameHeight

	// Parse hex color
	col := parseHexColor(cam.Color)

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Fill background with camera color
	draw.Draw(img, img.Bounds(), &image.Uniform{col}, image.Point{}, draw.Src)

	// Bottom black bar (60px)
	black := color.RGBA{0, 0, 0, 255}
	for y := H - 60; y < H; y++ {
		for x := 0; x < W; x++ {
			img.SetRGBA(x, y, black)
		}
	}

	// Timestamp
	ts := time.Now().Format("2006-01-02 15:04:05")
	drawText(img, 12, 20, ts, color.RGBA{255, 255, 255, 255})

	// Camera label
	label := fmt.Sprintf("%s [%s]", cam.Label, cameraID)
	drawText(img, 12, 38, label, color.RGBA{200, 230, 255, 255})

	// Door frame
	doorX := W/2 - 50
	doorY := H / 4
	doorW := 100
	doorH := H / 2
	doorCol := color.RGBA{255, 255, 255, 120}
	drawRect(img, doorX, doorY, doorX+doorW, doorY+doorH, doorCol, 2)

	// Action indicator
	switch action {
	case "entry":
		// Green person (ellipse + body)
		green := color.RGBA{0x2e, 0xcc, 0x71, 255}
		greenDark := color.RGBA{0x27, 0xae, 0x60, 255}
		fillEllipse(img, doorX+30, doorY+20, doorX+70, doorY+60, green, greenDark)
		fillRect(img, doorX+40, doorY+60, doorX+60, doorY+120, green, greenDark)
		drawText(img, doorX-20, doorY+doorH+20, "→  进入", color.RGBA{0x2e, 0xcc, 0x71, 255})

	case "exit":
		// Red person
		red := color.RGBA{0xe7, 0x4c, 0x3c, 255}
		redDark := color.RGBA{0xc0, 0x39, 0x2b, 255}
		fillEllipse(img, doorX-10, doorY+20, doorX+30, doorY+60, red, redDark)
		fillRect(img, doorX, doorY+60, doorX+20, doorY+120, red, redDark)
		drawText(img, doorX-20, doorY+doorH+20, "←  离开", color.RGBA{0xe7, 0x4c, 0x3c, 255})

	default: // idle
		drawText(img, doorX, doorY+doorH+20, "● 无人", color.RGBA{0x95, 0xa5, 0xa6, 255})
	}

	if person != "" {
		drawText(img, 12, H-48, fmt.Sprintf("Person: %s", person), color.RGBA{0xec, 0xf0, 0xf1, 255})
		if action != "idle" {
			drawText(img, 12, H-28, fmt.Sprintf("Action: %s", action), color.RGBA{0xf1, 0xc4, 0x0f, 255})
		}
	}

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: cfg.JPEGQuality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseHexColor converts a hex color string (e.g. "#2980b9") to color.RGBA.
func parseHexColor(hex string) color.RGBA {
	if len(hex) == 0 {
		return color.RGBA{85, 85, 119, 255}
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.RGBA{85, 85, 119, 255}
	}
	r := hexPair(hex[0:2])
	g := hexPair(hex[2:4])
	b := hexPair(hex[4:6])
	return color.RGBA{r, g, b, 255}
}

func hexPair(s string) uint8 {
	var v uint8
	for _, c := range s {
		v *= 16
		switch {
		case c >= '0' && c <= '9':
			v += uint8(c - '0')
		case c >= 'a' && c <= 'f':
			v += uint8(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v += uint8(c - 'A' + 10)
		}
	}
	return v
}

// drawRect outlines a rectangle with the given color and border width.
func drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA, width int) {
	for w := 0; w < width; w++ {
		for x := x1 + w; x < x2-w; x++ {
			if y1+w >= 0 && y1+w < img.Bounds().Max.Y && x >= 0 && x < img.Bounds().Max.X {
				img.SetRGBA(x, y1+w, col)
			}
			if y2-w-1 >= 0 && y2-w-1 < img.Bounds().Max.Y && x >= 0 && x < img.Bounds().Max.X {
				img.SetRGBA(x, y2-w-1, col)
			}
		}
		for y := y1 + w; y < y2-w; y++ {
			if x1+w >= 0 && x1+w < img.Bounds().Max.X && y >= 0 && y < img.Bounds().Max.Y {
				img.SetRGBA(x1+w, y, col)
			}
			if x2-w-1 >= 0 && x2-w-1 < img.Bounds().Max.X && y >= 0 && y < img.Bounds().Max.Y {
				img.SetRGBA(x2-w-1, y, col)
			}
		}
	}
}

// fillRect fills a rectangle with the given fill color and outline.
func fillRect(img *image.RGBA, x1, y1, x2, y2 int, fill, outline color.RGBA) {
	for y := y1; y < y2 && y < img.Bounds().Max.Y; y++ {
		for x := x1; x < x2 && x < img.Bounds().Max.X; x++ {
			if x >= 0 && y >= 0 {
				img.SetRGBA(x, y, fill)
			}
		}
	}
	drawRect(img, x1, y1, x2, y2, outline, 1)
}

// fillEllipse draws a filled ellipse in the bounding box.
func fillEllipse(img *image.RGBA, x1, y1, x2, y2 int, fill, outline color.RGBA) {
	cx := float64(x1+x2) / 2.0
	cy := float64(y1+y2) / 2.0
	rx := float64(x2-x1) / 2.0
	ry := float64(y2-y1) / 2.0

	for y := y1; y <= y2 && y < img.Bounds().Max.Y; y++ {
		for x := x1; x <= x2 && x < img.Bounds().Max.X; x++ {
			if x >= 0 && y >= 0 {
				dx := (float64(x) - cx) / rx
				dy := (float64(y) - cy) / ry
				if dx*dx+dy*dy <= 1.0 {
					img.SetRGBA(x, y, fill)
				}
			}
		}
	}
	// Simple outline by drawing boundary pixels
	for y := y1; y <= y2 && y < img.Bounds().Max.Y; y++ {
		for x := x1; x <= x2 && x < img.Bounds().Max.X; x++ {
			if x >= 0 && y >= 0 {
				dx := (float64(x) - cx) / rx
				dy := (float64(y) - cy) / ry
				dist := dx*dx + dy*dy
				if dist > 0.85 && dist <= 1.0 {
					img.SetRGBA(x, y, outline)
				}
			}
		}
	}
}
