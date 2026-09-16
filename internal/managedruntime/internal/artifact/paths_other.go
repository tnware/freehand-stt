//go:build !windows

package artifact

// IsReparse reports Windows link-like filesystem entries, including junctions.
func IsReparse(string) bool { return false }
