package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestPrepareBytesResizesJPEG(t *testing.T) {
	data := encodeTestJPEG(t, 200, 100)
	policy := DefaultPolicy(ModeFast)
	policy.MaxLongEdge = 50

	result, err := PrepareBytes(data, policy)
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if result.MimeType != "image/jpeg" {
		t.Fatalf("MimeType = %q, want image/jpeg", result.MimeType)
	}
	if result.Meta.Width != 50 || result.Meta.Height != 25 {
		t.Fatalf("sent size = %dx%d, want 50x25", result.Meta.Width, result.Meta.Height)
	}
	if result.Meta.OriginalWidth != 200 || result.Meta.OriginalHeight != 100 {
		t.Fatalf("original size = %dx%d, want 200x100", result.Meta.OriginalWidth, result.Meta.OriginalHeight)
	}
	if !result.Meta.Resized {
		t.Fatal("expected Resized")
	}
	if result.Meta.Scale <= 0 || result.Meta.Scale >= 1 {
		t.Fatalf("Scale = %f, want between 0 and 1", result.Meta.Scale)
	}
}

func TestPrepareBytesRawPreservesPNG(t *testing.T) {
	data := encodeTestPNG(t, 12, 8)
	result, err := PrepareBytes(data, DefaultPolicy(ModeRaw))
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if !bytes.Equal(result.Data, data) {
		t.Fatal("raw mode changed image data")
	}
	if result.MimeType != "image/png" {
		t.Fatalf("MimeType = %q, want image/png", result.MimeType)
	}
	if result.Meta.Width != 12 || result.Meta.Height != 8 {
		t.Fatalf("size = %dx%d, want 12x8", result.Meta.Width, result.Meta.Height)
	}
}

func TestPrepareBytesRejectsPixelLimit(t *testing.T) {
	data := encodeTestPNG(t, 10, 10)
	policy := DefaultPolicy(ModeAuto)
	policy.MaxPixels = 50
	_, err := PrepareBytes(data, policy)
	if err == nil {
		t.Fatal("PrepareBytes() error = nil, want pixel limit error")
	}
}

func encodeTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage(width, height)); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func encodeTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testImage(width, height), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("jpeg.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func testImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	return img
}
