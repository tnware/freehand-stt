package config

import (
	"errors"
	"fmt"
)

const (
	MinOverlaySizePercent    = 75
	MaxOverlaySizePercent    = 150
	MinOverlayOpacityPercent = 40
	MaxOverlayOpacityPercent = 100
	MinOverlayTopOffset      = 0
	MaxOverlayTopOffset      = 240
	MinOverlayGlowPercent    = 0
	MaxOverlayGlowPercent    = 100
)

type AppearanceMode string

type OverlayLayout string

type OverlayAnchor string

type OverlayVisibility string

type OverlayMotion string

type OverlaySurface string

type OverlayVisualizer string

const (
	AppearanceModeSystem AppearanceMode = "system"
	AppearanceModeLight  AppearanceMode = "light"
	AppearanceModeDark   AppearanceMode = "dark"
)

const (
	OverlayLayoutMinimal  OverlayLayout = "minimal"
	OverlayLayoutCapsule  OverlayLayout = "capsule"
	OverlayLayoutMeter    OverlayLayout = "meter"
	OverlayLayoutDetailed OverlayLayout = "detailed"
)

const (
	OverlayAnchorTopLeft      OverlayAnchor = "top-left"
	OverlayAnchorTopCenter    OverlayAnchor = "top-center"
	OverlayAnchorTopRight     OverlayAnchor = "top-right"
	OverlayAnchorBottomLeft   OverlayAnchor = "bottom-left"
	OverlayAnchorBottomCenter OverlayAnchor = "bottom-center"
	OverlayAnchorBottomRight  OverlayAnchor = "bottom-right"
)

const (
	OverlayVisibilityRecording OverlayVisibility = "recording"
	OverlayVisibilityActive    OverlayVisibility = "active"
	OverlayVisibilityAll       OverlayVisibility = "all"
)

const (
	OverlayMotionSystem  OverlayMotion = "system"
	OverlayMotionReduced OverlayMotion = "reduced"
)

const (
	OverlaySurfaceGlass   OverlaySurface = "glass"
	OverlaySurfaceSolid   OverlaySurface = "solid"
	OverlaySurfaceMinimal OverlaySurface = "minimal"
)

const (
	OverlayVisualizerBars     OverlayVisualizer = "bars"
	OverlayVisualizerPulse    OverlayVisualizer = "pulse"
	OverlayVisualizerEnvelope OverlayVisualizer = "envelope"
	OverlayVisualizerMeter    OverlayVisualizer = "meter"
)

// EffectiveAppearanceMode returns the colour mode that may be applied to the
// native windows and renderers. Windows owns Mica's light/dark appearance, so
// a saved solid-window preference is deliberately ignored while Mica is on.
func EffectiveAppearanceMode(useMica bool, mode AppearanceMode) AppearanceMode {
	if useMica {
		return AppearanceModeSystem
	}
	return mode
}

func (s Settings) EffectiveAppearanceMode() AppearanceMode {
	return EffectiveAppearanceMode(s.UseMica, s.AppearanceMode)
}

// OverlayPreferences is the renderer-safe, credential-free portion of Settings
// accepted by the native preview binding. It is deliberately narrower than a
// settings save request so previewing cannot mutate application configuration.
type OverlayPreferences struct {
	Layout         OverlayLayout     `json:"layout"`
	Anchor         OverlayAnchor     `json:"anchor"`
	Visibility     OverlayVisibility `json:"visibility"`
	Motion         OverlayMotion     `json:"motion"`
	Surface        OverlaySurface    `json:"surface"`
	Visualizer     OverlayVisualizer `json:"visualizer"`
	SizePercent    int               `json:"sizePercent"`
	OpacityPercent int               `json:"opacityPercent"`
	EdgeOffset     int               `json:"edgeOffset"`
	GlowPercent    int               `json:"glowPercent"`
}

func (s Settings) OverlayPreferences() OverlayPreferences {
	return OverlayPreferences{
		Layout: s.OverlayLayout, Anchor: s.OverlayAnchor, Visibility: s.OverlayVisibility,
		Motion: s.OverlayMotion, Surface: s.OverlaySurface, Visualizer: s.OverlayVisualizer,
		SizePercent: s.OverlaySizePercent, OpacityPercent: s.OverlayOpacityPercent,
		EdgeOffset: s.OverlayTopOffset, GlowPercent: s.OverlayGlowPercent,
	}
}

func ValidateOverlayPreferences(preferences OverlayPreferences) error {
	switch preferences.Layout {
	case OverlayLayoutMinimal, OverlayLayoutCapsule, OverlayLayoutMeter, OverlayLayoutDetailed:
	default:
		return errors.New("overlay layout is invalid")
	}
	switch preferences.Anchor {
	case OverlayAnchorTopLeft, OverlayAnchorTopCenter, OverlayAnchorTopRight,
		OverlayAnchorBottomLeft, OverlayAnchorBottomCenter, OverlayAnchorBottomRight:
	default:
		return errors.New("overlay anchor is invalid")
	}
	switch preferences.Visibility {
	case OverlayVisibilityRecording, OverlayVisibilityActive, OverlayVisibilityAll:
	default:
		return errors.New("overlay visibility is invalid")
	}
	switch preferences.Motion {
	case OverlayMotionSystem, OverlayMotionReduced:
	default:
		return errors.New("overlay motion is invalid")
	}
	switch preferences.Surface {
	case OverlaySurfaceGlass, OverlaySurfaceSolid, OverlaySurfaceMinimal:
	default:
		return errors.New("overlay surface is invalid")
	}
	switch preferences.Visualizer {
	case OverlayVisualizerBars, OverlayVisualizerPulse, OverlayVisualizerEnvelope, OverlayVisualizerMeter:
	default:
		return errors.New("overlay visualizer is invalid")
	}
	if preferences.SizePercent < MinOverlaySizePercent || preferences.SizePercent > MaxOverlaySizePercent {
		return fmt.Errorf("overlay size must be between %d and %d percent", MinOverlaySizePercent, MaxOverlaySizePercent)
	}
	if preferences.OpacityPercent < MinOverlayOpacityPercent || preferences.OpacityPercent > MaxOverlayOpacityPercent {
		return fmt.Errorf("overlay opacity must be between %d and %d percent", MinOverlayOpacityPercent, MaxOverlayOpacityPercent)
	}
	if preferences.EdgeOffset < MinOverlayTopOffset || preferences.EdgeOffset > MaxOverlayTopOffset {
		return fmt.Errorf("overlay edge distance must be between %d and %d pixels", MinOverlayTopOffset, MaxOverlayTopOffset)
	}
	if preferences.GlowPercent < MinOverlayGlowPercent || preferences.GlowPercent > MaxOverlayGlowPercent {
		return fmt.Errorf("overlay glow must be between %d and %d percent", MinOverlayGlowPercent, MaxOverlayGlowPercent)
	}
	return nil
}
