package inference

import "github.com/tnware/freehand-stt/internal/compatibility"

// WithCleanupOptions preserves the job's settings without mutating shared clients.
func (c *Client) WithCleanupOptions(options compatibility.CleanupOptions) *Client {
	copy := *c
	copy.cleanupOptions = options
	return &copy
}
