//go:build windows

package audio

import (
	"context"
	"github.com/gen2brain/malgo"
)

const nativeAudioBackend = malgo.BackendWasapi

func authorizeMicrophone(ctx context.Context, _ bool) error { return ctx.Err() }
