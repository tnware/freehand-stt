package managedruntime

import "testing"

func newTestWorker(t *testing.T, autoStart bool) *worker {
	t.Helper()
	return newWorker(t.TempDir(), providers[NeMoSpeechCPP], workerConfig{
		Model: "nemotron-3.5", Realtime: true, Enabled: autoStart, AutoStart: autoStart,
	}, nil, nil)
}

func resolveWorker(w *worker) (Endpoint, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.resolveLocked()
}

func configureWorker(w *worker, cfg workerConfig) {
	w.mu.Lock()
	process := w.configureLocked(cfg)
	w.mu.Unlock()
	if process != nil {
		process.Kill()
	}
}
