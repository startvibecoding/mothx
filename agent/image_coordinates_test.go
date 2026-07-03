package agent

import "testing"

func TestImageContentMapPointToOriginalWithCrop(t *testing.T) {
	img := ImageContent{
		Width:          100,
		Height:         50,
		OriginalWidth:  400,
		OriginalHeight: 300,
		Cropped:        true,
		CropX:          40,
		CropY:          30,
		CropWidth:      200,
		CropHeight:     100,
	}
	x, y, ok := img.MapPointToOriginal(50, 25)
	if !ok {
		t.Fatal("MapPointToOriginal() ok = false")
	}
	if x != 140 || y != 80 {
		t.Fatalf("point = %.1f,%.1f, want 140,80", x, y)
	}
}

func TestImageContentMapNormalizedRectToOriginal(t *testing.T) {
	img := ImageContent{
		Width:          100,
		Height:         50,
		OriginalWidth:  200,
		OriginalHeight: 100,
	}
	x, y, w, h, ok := img.MapNormalizedRectToOriginal(100, 200, 300, 400, 1000)
	if !ok {
		t.Fatal("MapNormalizedRectToOriginal() ok = false")
	}
	if x != 20 || y != 20 || w != 60 || h != 40 {
		t.Fatalf("rect = %.1f,%.1f %.1fx%.1f, want 20,20 60x40", x, y, w, h)
	}
}
