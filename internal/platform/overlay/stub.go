//go:build !windows && !darwin

package overlay

import (
	"errors"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

type StatusOverlay struct{}

func NewStatusOverlay() (*StatusOverlay, error)       { return nil, unavailable }
func (*StatusOverlay) Update(OverlayStatus) error     { return unavailable }
func (*StatusOverlay) Configure(OverlayOptions) error { return unavailable }
func (*StatusOverlay) Close() error                   { return nil }
func (*StatusOverlay) SetLevelSource(LevelSource)     {}
