package platform

import "strings"

// The native message loop owns this transient cache. Text measurement happens
// only when the caption or geometry changes, not on every animation frame.
type overlayCaptionCache struct {
	text, fitted string
	width, scale float64
}

func (c *overlayCaptionCache) fit(text string, width, scale float64, measure func(string) float64) string {
	if text == c.text && width == c.width && scale == c.scale {
		return c.fitted
	}
	runes := []rune(text)
	low, high := 0, len(runes)
	for low < high {
		mid := (low + high) / 2
		if measure(string(runes[mid:])) <= width {
			high = mid
		} else {
			low = mid + 1
		}
	}
	fitted := strings.TrimSpace(string(runes[low:]))
	*c = overlayCaptionCache{text: text, fitted: fitted, width: width, scale: scale}
	return fitted
}
