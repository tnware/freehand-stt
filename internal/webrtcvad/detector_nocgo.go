//go:build !cgo

package webrtcvad

import "errors"

const (
	ModeQuality        = 0
	ModeLowBitrate     = 1
	ModeAggressive     = 2
	ModeVeryAggressive = 3
)

type Detector struct{}

func New(int, int) (*Detector, error) {
	return nil, errors.New("silence-aware splitting requires a cgo-enabled build")
}

func (*Detector) Speech([]int16) (bool, error) { return false, errors.New("WebRTC VAD is unavailable") }
func (*Detector) Close() error                 { return nil }
