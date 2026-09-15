package modelprofile

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestNeMoControlAdmissionUsesQualifiedContract(t *testing.T) {
	for _, id := range []ID{Nemotron35, ParakeetTDT} {
		for _, delay := range []int{0, 100, 1200, 10000} {
			o := compatibility.TranscriptionOptions{NeMo: compatibility.NeMoOptions{Normalize: true, ProfanityFilter: true, EndpointingMilliseconds: delay}}
			if err := ValidateTranscription(id, compatibility.NeMoSpeechV1, "auto", o); err != nil {
				t.Fatalf("qualified controls: %v", err)
			}
			if ValidateTranscription(id, compatibility.Generic, "auto", o) == nil || ValidateTranscription(Generic, compatibility.NeMoSpeechV1, "auto", o) == nil {
				t.Fatal("unqualified backend/model admitted NeMo controls")
			}
		}
		for _, delay := range []int{-1, 1, 99, 10001} {
			o := compatibility.TranscriptionOptions{NeMo: compatibility.NeMoOptions{EndpointingMilliseconds: delay}}
			if ValidateTranscription(id, compatibility.NeMoSpeechV1, "auto", o) == nil || ValidateStoredTranscription(id, compatibility.NeMoSpeechV1, "auto", o) == nil {
				t.Fatal("invalid endpointing value admitted")
			}
		}
	}
}
