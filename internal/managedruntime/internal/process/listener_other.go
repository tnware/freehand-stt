//go:build (!windows && !darwin) || (darwin && !cgo)

package process

import "errors"

func OwnsListener(port, pid int) (bool, error) {
	return false, errors.New("managed listener ownership is unavailable in this build")
}
