//go:build !windows

package managedruntime

import (
	"errors"
	"os/exec"
)

func startInJob(*exec.Cmd) (func(), error) {
	return nil, errors.New("Managed speech requires Windows x86-64.")
}
