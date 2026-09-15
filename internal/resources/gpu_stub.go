//go:build (!windows && !darwin) || (darwin && !cgo)

package resources

func newGPUReader() gpuReader { return nil }
