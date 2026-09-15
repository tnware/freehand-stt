package inference

import (
	"mime/multipart"
	"strconv"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func writeNeMoOptions(w *multipart.Writer, o compatibility.NeMoOptions) error {
	for _, field := range []struct {
		name  string
		value bool
	}{
		{"automatic_punctuation", !o.DisablePunctuation},
		{"verbatim", !o.Normalize},
		{"profanity_filter", o.ProfanityFilter},
	} {
		if err := w.WriteField(field.name, strconv.FormatBool(field.value)); err != nil {
			return err
		}
	}
	return nil
}
