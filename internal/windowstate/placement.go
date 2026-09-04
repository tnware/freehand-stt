// Package windowstate owns durable native window placement and the geometry
// rules used to keep Freehand's windows visible across changing displays.
package windowstate

import "github.com/wailsapp/wails/v3/pkg/application"

// Placement stores a window's normal bounds relative to a screen work area.
// ScreenName is retained alongside ScreenID because a Windows monitor handle
// may change between sessions while the display device name remains stable.
type Placement struct {
	ScreenID   string `json:"screenID,omitempty"`
	ScreenName string `json:"screenName,omitempty"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// Capture converts absolute window bounds into work-area-relative placement.
func Capture(bounds application.Rect, screen *application.Screen) (Placement, bool) {
	if screen == nil || bounds.Width <= 0 || bounds.Height <= 0 || screen.WorkArea.Width <= 0 || screen.WorkArea.Height <= 0 {
		return Placement{}, false
	}
	return Placement{
		ScreenID:   screen.ID,
		ScreenName: screen.Name,
		X:          bounds.X - screen.WorkArea.X,
		Y:          bounds.Y - screen.WorkArea.Y,
		Width:      bounds.Width,
		Height:     bounds.Height,
	}, true
}

// Resolve returns safe absolute bounds for a saved placement. If its original
// display disappeared, the window is centered on the primary display instead.
func Resolve(saved Placement, screens []*application.Screen, defaultWidth, defaultHeight, minWidth, minHeight int) (application.Rect, *application.Screen, bool) {
	if saved.Width <= 0 || saved.Height <= 0 {
		return application.Rect{}, nil, false
	}

	screen, matched := savedScreen(saved, screens)
	if screen == nil || screen.WorkArea.Width <= 0 || screen.WorkArea.Height <= 0 {
		return application.Rect{}, nil, false
	}

	workArea := screen.WorkArea
	width := boundedDimension(saved.Width, defaultWidth, minWidth, workArea.Width)
	height := boundedDimension(saved.Height, defaultHeight, minHeight, workArea.Height)
	x := workArea.X + saved.X
	y := workArea.Y + saved.Y
	if !matched {
		x = workArea.X + (workArea.Width-width)/2
		y = workArea.Y + (workArea.Height-height)/2
	}

	return Clamp(application.Rect{X: x, Y: y, Width: width, Height: height}, workArea), screen, true
}

// CenterOver centers a transient window over the main window and keeps it in
// the current display's usable work area.
func CenterOver(owner application.Rect, childWidth, childHeight int, workArea application.Rect) application.Rect {
	childWidth = boundedDimension(childWidth, childWidth, 1, workArea.Width)
	childHeight = boundedDimension(childHeight, childHeight, 1, workArea.Height)
	return Clamp(application.Rect{
		X:      owner.X + (owner.Width-childWidth)/2,
		Y:      owner.Y + (owner.Height-childHeight)/2,
		Width:  childWidth,
		Height: childHeight,
	}, workArea)
}

// Clamp keeps a complete rectangle inside a work area.
func Clamp(bounds, workArea application.Rect) application.Rect {
	if workArea.Width <= 0 || workArea.Height <= 0 {
		return bounds
	}
	bounds.Width = min(max(bounds.Width, 1), workArea.Width)
	bounds.Height = min(max(bounds.Height, 1), workArea.Height)
	bounds.X = min(max(bounds.X, workArea.X), workArea.X+workArea.Width-bounds.Width)
	bounds.Y = min(max(bounds.Y, workArea.Y), workArea.Y+workArea.Height-bounds.Height)
	return bounds
}

func savedScreen(saved Placement, screens []*application.Screen) (*application.Screen, bool) {
	for _, screen := range screens {
		if screen != nil && saved.ScreenID != "" && screen.ID == saved.ScreenID {
			return screen, true
		}
	}
	for _, screen := range screens {
		if screen != nil && saved.ScreenName != "" && screen.Name == saved.ScreenName {
			return screen, true
		}
	}
	for _, screen := range screens {
		if screen != nil && screen.IsPrimary {
			return screen, false
		}
	}
	for _, screen := range screens {
		if screen != nil {
			return screen, false
		}
	}
	return nil, false
}

func boundedDimension(value, fallback, minimum, maximum int) int {
	if maximum <= 0 {
		return value
	}
	if value <= 0 {
		value = fallback
	}
	minimum = min(max(minimum, 1), maximum)
	return min(max(value, minimum), maximum)
}
