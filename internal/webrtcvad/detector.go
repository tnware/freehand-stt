//go:build cgo

package webrtcvad

/*
#cgo CFLAGS: -std=c11 -I${SRCDIR}/upstream/include -I${SRCDIR}/upstream/src
#include "fvad.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

const (
	ModeQuality        = 0
	ModeLowBitrate     = 1
	ModeAggressive     = 2
	ModeVeryAggressive = 3
)

type Detector struct {
	instance *C.Fvad
}

func New(sampleRate, mode int) (*Detector, error) {
	instance := C.fvad_new()
	if instance == nil {
		return nil, errors.New("WebRTC VAD allocation failed")
	}
	detector := &Detector{instance: instance}
	if C.fvad_set_sample_rate(instance, C.int(sampleRate)) != 0 {
		detector.Close()
		return nil, errors.New("WebRTC VAD sample rate is unsupported")
	}
	if C.fvad_set_mode(instance, C.int(mode)) != 0 {
		detector.Close()
		return nil, errors.New("WebRTC VAD mode is invalid")
	}
	runtime.SetFinalizer(detector, (*Detector).Close)
	return detector, nil
}

func (d *Detector) Speech(samples []int16) (bool, error) {
	if d == nil || d.instance == nil {
		return false, errors.New("WebRTC VAD is closed")
	}
	if len(samples) == 0 {
		return false, errors.New("WebRTC VAD frame is empty")
	}
	result := C.fvad_process(
		d.instance,
		(*C.int16_t)(unsafe.Pointer(unsafe.SliceData(samples))),
		C.size_t(len(samples)),
	)
	runtime.KeepAlive(samples)
	if result < 0 {
		return false, errors.New("WebRTC VAD frame length is unsupported")
	}
	return result == 1, nil
}

func (d *Detector) Close() error {
	if d == nil || d.instance == nil {
		return nil
	}
	C.fvad_free(d.instance)
	d.instance = nil
	runtime.SetFinalizer(d, nil)
	return nil
}
