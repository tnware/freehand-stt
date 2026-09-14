package managedruntime

import "golang.org/x/sys/unix"

func hostOSVersion() uint32 {
	value, err := unix.Sysctl("kern.osproductversion")
	if err != nil {
		return 0
	}
	return macOSVersion(value)
}
