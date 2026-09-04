//go:build cgo

package webrtcvad

import "testing"

func TestDetectorAcceptsSilenceFrame(t *testing.T) {
	detector, err := New(16000, ModeAggressive)
	if err != nil {
		t.Fatal(err)
	}
	defer detector.Close()

	speech, err := detector.Speech(make([]int16, 320))
	if err != nil {
		t.Fatal(err)
	}
	if speech {
		t.Fatal("digital silence was classified as speech")
	}
}

func TestDetectorRejectsInvalidConfigurationAndFrame(t *testing.T) {
	if _, err := New(12345, ModeAggressive); err == nil {
		t.Fatal("invalid sample rate was accepted")
	}
	detector, err := New(16000, ModeAggressive)
	if err != nil {
		t.Fatal(err)
	}
	defer detector.Close()
	if _, err := detector.Speech(make([]int16, 17)); err == nil {
		t.Fatal("invalid frame was accepted")
	}
}
