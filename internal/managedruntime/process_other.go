//go:build !windows && !darwin

package managedruntime

import (
	"errors"
	"os/exec"
)

func startOwnedProcess(*exec.Cmd) (func(), int, error) {
	return nil, 0, errors.New("Managed runtimes are unavailable on this platform.")
}
