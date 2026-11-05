//go:build !windows && !darwin && !linux

package startup

import (
	"errors"
)

var unavailable = errors.New("This native feature is unavailable on this platform")

type Startup struct{}

func (Startup) Set(bool) error         { return unavailable }
func (Startup) Enabled() (bool, error) { return false, unavailable }
