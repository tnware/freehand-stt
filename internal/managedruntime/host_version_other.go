//go:build !darwin

package managedruntime

func hostOSVersion() uint32 { return 0 }
