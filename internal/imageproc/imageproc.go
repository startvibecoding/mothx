package imageproc

import (
	"bytes"
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"

	_ "image/gif"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type Mode string

const (
	ModeAuto   Mode = "auto"
	ModeFast   Mode = "fast"
	ModeDetail Mode = "detail"
	ModeRaw    Mode = "raw"
)

const (
	DefaultMaxFileBytes = 10 << 20
	DefaultMaxPixels    = 40_000_000
)

type Policy struct {
	Mode           Mode
	MaxFileBytes   int64
	MaxPixels      int
	MaxLongEdge    int
	MaxOutputBytes int
	Crop           *Crop
}

type Crop struct {
	X      int
	Y      int
	Width  int
	Height int
}

type Meta struct {
	Width          int
	Height         int
	Bytes          int
	OriginalWidth  int
	OriginalHeight int
	OriginalBytes  int
	Resized        bool
	Cropped        bool
	Transcoded     bool
	Scale          float64
	Detail         string
	CropX          int
	CropY          int
	CropWidth      int
	CropHeight     int
}

type Result struct {
	Data     []byte
	MimeType string
	Meta     Meta
}

func NormalizeMode(s string) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case ModeFast:
		return ModeFast
	case ModeDetail:
		return ModeDetail
	case ModeRaw:
		return ModeRaw
	default:
		return ModeAuto
	}
}

func DefaultPolicy(mode Mode) Policy {
	p := Policy{
		Mode:         mode,
		MaxFileBytes: DefaultMaxFileBytes,
		MaxPixels:    DefaultMaxPixels,
	}
	switch mode {
	case ModeFast:
		p.MaxLongEdge = 1024
		p.MaxOutputBytes = 2 << 20
	case ModeDetail:
		p.MaxLongEdge = 2048
		p.MaxOutputBytes = 6 << 20
	case ModeRaw:
		p.MaxOutputBytes = DefaultMaxFileBytes
	default:
		p.Mode = ModeAuto
		p.MaxLongEdge = 1568
		p.MaxOutputBytes = 3 << 20
	}
	return p
}

func PrepareFile(path string, policy Policy) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	return PrepareBytes(data, policy)
}

func PrepareBytes(data []byte, policy Policy) (Result, error) {
	policy = normalizePolicy(policy)
	if policy.MaxFileBytes > 0 && int64(len(data)) > policy.MaxFileBytes {
		return Result{}, fmt.Errorf("image file too large: %d bytes (max %d)", len(data), policy.MaxFileBytes)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Result{}, fmt.Errorf("inspect image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Result{}, fmt.Errorf("inspect image: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if policy.MaxPixels > 0 && pixels > int64(policy.MaxPixels) {
		return Result{}, fmt.Errorf("image has too many pixels: %d (max %d)", pixels, policy.MaxPixels)
	}

	sourceMime := mimeFromFormat(format)
	meta := Meta{
		Width:          cfg.Width,
		Height:         cfg.Height,
		Bytes:          len(data),
		OriginalWidth:  cfg.Width,
		OriginalHeight: cfg.Height,
		OriginalBytes:  len(data),
		Scale:          1,
		Detail:         string(policy.Mode),
	}

	if policy.Mode == ModeRaw && policy.Crop == nil {
		return Result{Data: data, MimeType: sourceMime, Meta: meta}, nil
	}

	cropRect, needsCrop, err := normalizedCrop(policy.Crop, cfg.Width, cfg.Height)
	if err != nil {
		return Result{}, err
	}

	sourceW, sourceH := cfg.Width, cfg.Height
	if needsCrop {
		sourceW = cropRect.Dx()
		sourceH = cropRect.Dy()
		meta.Width = sourceW
		meta.Height = sourceH
		meta.Cropped = true
		meta.CropX = cropRect.Min.X
		meta.CropY = cropRect.Min.Y
		meta.CropWidth = sourceW
		meta.CropHeight = sourceH
	}

	targetW, targetH, scale := scaledDimensions(sourceW, sourceH, policy.MaxLongEdge)
	needsResize := targetW != sourceW || targetH != sourceH
	needsTranscode := sourceMime != "image/png" && sourceMime != "image/jpeg"
	tooLarge := policy.MaxOutputBytes > 0 && len(data) > policy.MaxOutputBytes
	if !needsCrop && !needsResize && !needsTranscode && !tooLarge {
		return Result{Data: data, MimeType: sourceMime, Meta: meta}, nil
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Result{}, fmt.Errorf("decode image: %w", err)
	}
	if needsCrop {
		img = cropImage(img, cropRect)
	}
	if needsResize {
		img = resizeImage(img, targetW, targetH)
		meta.Width = targetW
		meta.Height = targetH
		meta.Resized = true
		meta.Scale = scale
	}

	out, mimeType, err := encodeForPolicy(img, sourceMime, policy)
	if err != nil {
		return Result{}, err
	}
	meta.Bytes = len(out)
	meta.Transcoded = mimeType != sourceMime || format != formatFromMime(mimeType)
	return Result{Data: out, MimeType: mimeType, Meta: meta}, nil
}

func normalizePolicy(policy Policy) Policy {
	mode := policy.Mode
	if mode == "" {
		mode = ModeAuto
	}
	base := DefaultPolicy(mode)
	if policy.MaxFileBytes > 0 {
		base.MaxFileBytes = policy.MaxFileBytes
	}
	if policy.MaxPixels > 0 {
		base.MaxPixels = policy.MaxPixels
	}
	if policy.MaxLongEdge > 0 {
		base.MaxLongEdge = policy.MaxLongEdge
	}
	if policy.MaxOutputBytes > 0 {
		base.MaxOutputBytes = policy.MaxOutputBytes
	}
	if policy.Crop != nil {
		crop := *policy.Crop
		base.Crop = &crop
	}
	return base
}

func scaledDimensions(width, height, maxLongEdge int) (int, int, float64) {
	if maxLongEdge <= 0 || (width <= maxLongEdge && height <= maxLongEdge) {
		return width, height, 1
	}
	longEdge := width
	if height > longEdge {
		longEdge = height
	}
	scale := float64(maxLongEdge) / float64(longEdge)
	w := int(math.Round(float64(width) * scale))
	h := int(math.Round(float64(height) * scale))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h, scale
}

func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	imagedraw.Draw(dst, dst.Bounds(), src, rect.Min, imagedraw.Src)
	return dst
}

func normalizedCrop(crop *Crop, width, height int) (image.Rectangle, bool, error) {
	if crop == nil {
		return image.Rectangle{}, false, nil
	}
	if crop.Width <= 0 || crop.Height <= 0 {
		return image.Rectangle{}, false, fmt.Errorf("invalid crop: width and height must be positive")
	}
	if crop.X < 0 || crop.Y < 0 {
		return image.Rectangle{}, false, fmt.Errorf("invalid crop: x and y must be non-negative")
	}
	rect := image.Rect(crop.X, crop.Y, crop.X+crop.Width, crop.Y+crop.Height)
	bounds := image.Rect(0, 0, width, height)
	if !rect.In(bounds) {
		return image.Rectangle{}, false, fmt.Errorf("invalid crop: rectangle %dx%d+%d+%d exceeds image bounds %dx%d", crop.Width, crop.Height, crop.X, crop.Y, width, height)
	}
	if rect.Eq(bounds) {
		return image.Rectangle{}, false, nil
	}
	return rect, true, nil
}

func encodeForPolicy(img image.Image, sourceMime string, policy Policy) ([]byte, string, error) {
	if hasTransparency(img) {
		data, err := encodePNG(img)
		if err != nil {
			return nil, "", err
		}
		return data, "image/png", nil
	}

	if sourceMime == "image/png" {
		data, err := encodePNG(img)
		if err != nil {
			return nil, "", err
		}
		if policy.MaxOutputBytes <= 0 || len(data) <= policy.MaxOutputBytes {
			return data, "image/png", nil
		}
	}

	for _, quality := range []int{90, 85, 80, 75, 70} {
		data, err := encodeJPEG(img, quality)
		if err != nil {
			return nil, "", err
		}
		if policy.MaxOutputBytes <= 0 || len(data) <= policy.MaxOutputBytes || quality == 70 {
			return data, "image/jpeg", nil
		}
	}
	return nil, "", fmt.Errorf("encode image: no encoder selected")
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				return true
			}
		}
	}
	return false
}

func mimeFromFormat(format string) string {
	switch strings.ToLower(format) {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func formatFromMime(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return ""
	}
}
