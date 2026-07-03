package provider

// MapPointToOriginal maps a point in the sent image coordinate space back to
// the original source image coordinate space.
func (img ImageContent) MapPointToOriginal(x, y float64) (float64, float64, bool) {
	if img.Width <= 0 || img.Height <= 0 || img.OriginalWidth <= 0 || img.OriginalHeight <= 0 {
		return 0, 0, false
	}
	sourceW := img.OriginalWidth
	sourceH := img.OriginalHeight
	offsetX := 0
	offsetY := 0
	if img.Cropped {
		if img.CropWidth <= 0 || img.CropHeight <= 0 {
			return 0, 0, false
		}
		sourceW = img.CropWidth
		sourceH = img.CropHeight
		offsetX = img.CropX
		offsetY = img.CropY
	}
	scaleX := float64(sourceW) / float64(img.Width)
	scaleY := float64(sourceH) / float64(img.Height)
	return float64(offsetX) + x*scaleX, float64(offsetY) + y*scaleY, true
}

// MapRectToOriginal maps a rectangle in sent image coordinates back to the
// original source image coordinate space.
func (img ImageContent) MapRectToOriginal(x, y, width, height float64) (float64, float64, float64, float64, bool) {
	x1, y1, ok := img.MapPointToOriginal(x, y)
	if !ok {
		return 0, 0, 0, 0, false
	}
	x2, y2, ok := img.MapPointToOriginal(x+width, y+height)
	if !ok {
		return 0, 0, 0, 0, false
	}
	return x1, y1, x2 - x1, y2 - y1, true
}

// MapNormalizedPointToOriginal maps a point from a normalized coordinate space
// such as [0,1000] back to the original source image coordinate space.
func (img ImageContent) MapNormalizedPointToOriginal(x, y, max float64) (float64, float64, bool) {
	if max <= 0 || img.Width <= 0 || img.Height <= 0 {
		return 0, 0, false
	}
	return img.MapPointToOriginal(x/max*float64(img.Width), y/max*float64(img.Height))
}

// MapNormalizedRectToOriginal maps a rectangle from a normalized coordinate
// space such as [0,1000] back to the original source image coordinate space.
func (img ImageContent) MapNormalizedRectToOriginal(x, y, width, height, max float64) (float64, float64, float64, float64, bool) {
	if max <= 0 || img.Width <= 0 || img.Height <= 0 {
		return 0, 0, 0, 0, false
	}
	return img.MapRectToOriginal(
		x/max*float64(img.Width),
		y/max*float64(img.Height),
		width/max*float64(img.Width),
		height/max*float64(img.Height),
	)
}
