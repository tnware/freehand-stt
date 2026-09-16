package overlay

// LevelSource supplies capture amplitude to a meter. It is deliberately the
// narrowest possible surface: one scalar, taken and cleared, carrying no audio
// and no transcript content.
type LevelSource interface {
	// TakeLevel returns the peak amplitude since the previous call, 0..1.
	TakeLevel() float64
}
