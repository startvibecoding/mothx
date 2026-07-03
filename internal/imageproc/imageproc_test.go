package imageproc

import (
	"bytes"
	"encoding/base64"
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

func TestPrepareBytesCropsImage(t *testing.T) {
	data := encodeTestPNG(t, 120, 80)
	policy := DefaultPolicy(ModeDetail)
	policy.Crop = &Crop{X: 10, Y: 12, Width: 40, Height: 20}

	result, err := PrepareBytes(data, policy)
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if result.Meta.Width != 40 || result.Meta.Height != 20 {
		t.Fatalf("sent size = %dx%d, want 40x20", result.Meta.Width, result.Meta.Height)
	}
	if !result.Meta.Cropped {
		t.Fatal("expected Cropped")
	}
	if result.Meta.CropX != 10 || result.Meta.CropY != 12 || result.Meta.CropWidth != 40 || result.Meta.CropHeight != 20 {
		t.Fatalf("crop meta = %+v, want 40x20+10+12", result.Meta)
	}
	if result.Meta.OriginalWidth != 120 || result.Meta.OriginalHeight != 80 {
		t.Fatalf("original size = %dx%d, want 120x80", result.Meta.OriginalWidth, result.Meta.OriginalHeight)
	}
}

func TestPrepareBytesRejectsOutOfBoundsCrop(t *testing.T) {
	data := encodeTestPNG(t, 20, 20)
	policy := DefaultPolicy(ModeAuto)
	policy.Crop = &Crop{X: 10, Y: 10, Width: 20, Height: 20}
	_, err := PrepareBytes(data, policy)
	if err == nil {
		t.Fatal("PrepareBytes() error = nil, want crop bounds error")
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

func TestPrepareBytesResizesToOutputLimit(t *testing.T) {
	data := encodeNoisyJPEG(t, 800, 600)
	policy := DefaultPolicy(ModeDetail)
	policy.MaxLongEdge = 800
	policy.MaxOutputBytes = 25 * 1024

	result, err := PrepareBytes(data, policy)
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if result.Meta.Bytes > policy.MaxOutputBytes {
		t.Fatalf("encoded bytes = %d, want <= %d", result.Meta.Bytes, policy.MaxOutputBytes)
	}
	if !result.Meta.Resized {
		t.Fatal("expected Resized")
	}
	if result.Meta.Width >= 800 || result.Meta.Height >= 600 {
		t.Fatalf("size = %dx%d, want smaller than original", result.Meta.Width, result.Meta.Height)
	}
}

func TestPrepareBytesPreservesTransparentPNG(t *testing.T) {
	data := encodeTransparentPNG(t, 100, 80)
	policy := DefaultPolicy(ModeDetail)
	policy.MaxLongEdge = 50

	result, err := PrepareBytes(data, policy)
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if result.MimeType != "image/png" {
		t.Fatalf("MimeType = %q, want image/png", result.MimeType)
	}
	if result.Meta.Width != 50 || result.Meta.Height != 40 {
		t.Fatalf("size = %dx%d, want 50x40", result.Meta.Width, result.Meta.Height)
	}
}

func TestPrepareBytesDecodesWebP(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	result, err := PrepareBytes(data, DefaultPolicy(ModeAuto))
	if err != nil {
		t.Fatalf("PrepareBytes() error = %v", err)
	}
	if result.Meta.OriginalWidth != 1 || result.Meta.OriginalHeight != 1 {
		t.Fatalf("original size = %dx%d, want 1x1", result.Meta.OriginalWidth, result.Meta.OriginalHeight)
	}
	if result.MimeType != "image/jpeg" && result.MimeType != "image/png" {
		t.Fatalf("MimeType = %q, want image/jpeg or image/png", result.MimeType)
	}
	if result.MimeType == "image/webp" {
		t.Fatal("webp input should be transcoded for provider compatibility")
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

func encodeTransparentPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, transparentImage(width, height)); err != nil {
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

func encodeNoisyJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, noisyImage(width, height), &jpeg.Options{Quality: 95}); err != nil {
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

func noisyImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x*37 + y*17) % 256),
				G: uint8((x*13 + y*53) % 256),
				B: uint8((x*91 + y*29) % 256),
				A: 255,
			})
		}
	}
	return img
}

func transparentImage(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: 20, G: 120, B: 220, A: uint8(80 + (x+y)%120)})
		}
	}
	return img
}
