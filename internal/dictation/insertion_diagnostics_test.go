package dictation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
)

type diagnosticPlatform struct {
	platFake
	captureErr, validationErr, sendErr error
}

func (p *diagnosticPlatform) CaptureTarget() (insertion.Target, error) {
	if p.captureErr != nil {
		return insertion.Target{}, p.captureErr
	}
	return validTarget(), nil
}
func (p *diagnosticPlatform) Foreground() (insertion.Target, error) {
	return validTarget(), p.validationErr
}
func (p *diagnosticPlatform) InsertUnicode(ctx context.Context, target insertion.Target, text string) error {
	p.platFake.InsertUnicode(ctx, target, text)
	return p.sendErr
}
func diagnosticRecorder(p *diagnosticPlatform, manual bool, body *string) *testRecorder {
	cfg := config.Default()
	cfg.BaseURL = "https://stt.example/v1"
	cfg.Model = "speech"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.AutoInsert = !manual
	cfg.VADEnabled = false
	cfg.SilenceSplitting = false
	cfg.AutoStopEnabled = false
	cfg.PostProcessing.Enabled = false
	client := inference.New()
	client.HTTP.Transport = processingTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(*body))}, nil
	})
	return New(capFake{}, p, client, nil, staticSettings{value: cfg}, nil)
}

func TestRecorderInsertionDiagnosticDelivery(t *testing.T) {
	for _, mode := range []string{"capture", "validate", "send", "raw-capture", "raw-validate", "raw-send", "manual"} {
		t.Run(mode, func(t *testing.T) {
			p := &diagnosticPlatform{}
			reason := insertion.NewRejection(insertion.Capture, "value_not_settable")
			want := insertion.CopyRequiredMessage(reason)
			switch mode {
			case "capture", "manual":
				p.captureErr = reason
			case "validate":
				p.validationErr = insertion.NewRejection(insertion.Validate, "focused_element_changed")
				want = insertion.CopyRequiredMessage(p.validationErr)
			case "send":
				p.sendErr = insertion.NewRejection(insertion.Send, "modifiers_held")
				want = insertion.CopyRequiredMessage(p.sendErr)
			default:
				raw := errors.New("private AX app identity and field text")
				if mode == "raw-capture" {
					p.captureErr = raw
				} else if mode == "raw-validate" {
					p.validationErr = raw
				} else {
					p.sendErr = raw
				}
				want = insertion.CopyRequiredMessage(raw)
			}
			if mode == "manual" {
				want = "Transcript ready to copy"
			}
			body := `{"text":"test transcript"}`
			c := diagnosticRecorder(p, mode == "manual", &body)
			defer c.close()
			if err := c.StartWithMode(RecordingHold); err != nil {
				t.Fatal(err)
			}
			err := c.Stop()
			if mode == "manual" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, insertion.ErrCopyRequired) {
				t.Fatalf("sentinel: %v", err)
			}
			s := c.Status()
			if !s.CanCopy || s.Message != want || s.Transcript != "test transcript" || p.copies != 0 {
				t.Fatalf("status=%+v copies=%d", s, p.copies)
			}
			c.mu.Lock()
			retained := len(c.targets)
			c.mu.Unlock()
			if retained != 0 {
				t.Fatal("delivery retained capture reason/target")
			}
			// A subsequent healthy run must never inherit an earlier capture rejection.
			p.captureErr = nil
			p.validationErr = nil
			p.sendErr = nil
			if err := c.StartWithMode(RecordingHold); err != nil {
				t.Fatal(err)
			}
			if err := c.Stop(); err != nil {
				t.Fatal(err)
			}
			if mode != "manual" && (c.Status().CanCopy || c.Status().Message != "") {
				t.Fatalf("stale reason: %+v", c.Status())
			}
		})
	}
}
