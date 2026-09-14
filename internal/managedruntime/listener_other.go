//go:build (!windows && !darwin) || (darwin && !cgo)

package managedruntime

import "errors"

func ownsListener(port, pid int) (bool, error) {
	return false, errors.New("managed listener ownership is unavailable in this build")
}
