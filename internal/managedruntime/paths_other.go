//go:build !windows

package managedruntime

func isReparse(string) bool { return false }
