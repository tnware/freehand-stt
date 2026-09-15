//go:build (!windows && !darwin) || (darwin && !cgo)

package resources

func readCounters() counters { return counters{} }
