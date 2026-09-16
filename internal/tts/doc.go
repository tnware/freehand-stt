// Package tts owns explicit speech generation and one native playback session.
// Sources, generation, playback, and export share the service's admission,
// generation, and shutdown guards. Synthesized audio remains in Go.
package tts
