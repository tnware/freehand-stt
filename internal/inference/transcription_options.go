package inference

import (
	"mime/multipart"
	"strconv"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

// WithTranscriptionOptions captures request controls without mutating a shared
// client. All fields are values, so jobs and checkpoints keep their snapshot.
func (c *Client) WithTranscriptionOptions(options compatibility.TranscriptionOptions) *Client {
	copy := *c
	copy.transcriptionOptions = options
	return &copy
}

func (c *Client) validateTranscriptionOptions() error {
	if err := compatibility.ValidateTranscriptionOptions(c.profile, c.transcriptionOptions); err != nil {
		return &Error{Kind: "invalid_settings", Message: err.Error()}
	}
	return nil
}

func writeTranscriptionOptions(mw *multipart.Writer, options compatibility.TranscriptionOptions) error {
	if options.Prompt != "" {
		if err := mw.WriteField("prompt", options.Prompt); err != nil {
			return err
		}
	}
	if options.Hotwords != "" {
		if err := mw.WriteField("hotwords", options.Hotwords); err != nil {
			return err
		}
	}
	if options.TemperatureOverride {
		return mw.WriteField("temperature", strconv.FormatFloat(options.Temperature, 'f', -1, 64))
	}
	return nil
}
