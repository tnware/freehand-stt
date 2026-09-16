//go:build !windows && !darwin

package startup

import (
	"errors"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

type Startup struct{}

func (Startup) Set(bool) error         { return unavailable }
func (Startup) Enabled() (bool, error) { return false, unavailable }
