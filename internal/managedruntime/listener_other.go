//go:build !windows

package managedruntime

import "errors"

func ownsListener(port, pid int) (bool, error) {
	return false, errors.New("managed listener ownership is available only on Windows")
}
