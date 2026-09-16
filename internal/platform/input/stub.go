//go:build !windows && !darwin

package input

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/insertion"
	"log/slog"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

type Input struct{}

func NewInput(*slog.Logger) Input { return Input{} }

func (Input) CaptureTarget() (insertion.Target, error)                      { return insertion.Target{}, unavailable }
func (Input) Foreground() (insertion.Target, error)                         { return insertion.Target{}, unavailable }
func (Input) InsertUnicode(context.Context, insertion.Target, string) error { return unavailable }
func (Input) Copy(context.Context, string) error                            { return unavailable }
