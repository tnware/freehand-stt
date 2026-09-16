// Package realtime owns versioned speech transports. Shared session lifecycle
// dispatches to the distinct NeMo and vLLM wire protocols. It never captures
// audio, retains history, runs cleanup, or inserts text into another application.
package realtime
