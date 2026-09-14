package managedruntime

import (
	"context"
	"io"
	"time"
)

const acquisitionInterval = 200 * time.Millisecond

// Adapter callbacks run synchronously on the acquisition owner. Emit phase
// transitions and the first measured bytes immediately; coalesce steady transfer
// updates to five per second, with a final transfer update before verification.
type acquisitionReporter struct {
	emit func(AcquisitionProgress)
	last AcquisitionProgress
	at   time.Time
}

func (r *acquisitionReporter) update(p AcquisitionProgress) {
	p = boundedAcquisition(p)
	if r.emit == nil || p == r.last {
		return
	}
	now := time.Now()
	if p.Phase == r.last.Phase && r.last.Bytes > 0 && p.Bytes != p.TotalBytes && now.Sub(r.at) < acquisitionInterval {
		return
	}
	r.last, r.at = p, now
	r.emit(p)
}
func boundedAcquisition(p AcquisitionProgress) AcquisitionProgress {
	switch p.Phase {
	case "preparing", "downloading", "verifying":
	default:
		return AcquisitionProgress{Phase: "preparing"}
	}
	// Catalog model files are much smaller; this also keeps JSON integers exact.
	const maxBytes int64 = 1 << 40
	if p.TotalBytes < 0 || p.TotalBytes > maxBytes {
		p.TotalBytes = 0
	}
	if p.Bytes < 0 {
		p.Bytes = 0
	}
	if p.Bytes > maxBytes {
		p.Bytes = maxBytes
	}
	if p.TotalBytes > 0 && p.Bytes > p.TotalBytes {
		p.Bytes = p.TotalBytes
	}
	return p
}

type acquisitionWriter struct {
	io.Writer
	report       *acquisitionReporter
	bytes, total int64
}

func (w *acquisitionWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.bytes += int64(n)
	w.report.update(AcquisitionProgress{Phase: "downloading", Bytes: w.bytes, TotalBytes: w.total})
	return n, err
}
func (s *worker) acquisitionProgress(ctx context.Context, p AcquisitionProgress) {
	p = boundedAcquisition(p)
	s.mu.Lock()
	if s.closed || ctx.Err() != nil || !s.busy || s.status.Operation.Outcome != "running" || s.status.Operation.Kind != "download" {
		s.mu.Unlock()
		return
	}
	s.status.Acquisition = p
	s.status.Progress = -1
	if p.Phase == "downloading" && p.TotalBytes > 0 {
		s.status.Progress = float64(p.Bytes) / float64(p.TotalBytes)
	}
	// Preserve installing for older renderers; acquisition.phase distinguishes
	// preparation, transfer and verification without conflating 100% with success.
	s.mu.Unlock()
	s.notify()
}
