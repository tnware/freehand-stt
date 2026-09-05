package filetranscription

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"testing"
)

func TestStreamingObservationIsScopedToCompatibilityProfile(t *testing.T) {
	cfg := config.Default()
	cfg.BaseURL = "https://example.test/v1"
	cfg.Model = "same-model"
	service := &Service{fileStreamingUnsupported: map[fileStreamingCapabilityKey]string{fileStreamingKey(cfg): "incompatible_sse_contract"}}
	var status FileTranscriptionStatus
	service.applyFileStreamingCapabilityLocked(&status, cfg)
	if !status.StreamingUnavailable {
		t.Fatal("missing recorded observation")
	}
	cfg.CompatibilityProfile = compatibility.Speaches
	service.applyFileStreamingCapabilityLocked(&status, cfg)
	if status.StreamingUnavailable {
		t.Fatal("old profile disabled new profile")
	}
	cfg.CompatibilityProfile = ""
	service.applyFileStreamingCapabilityLocked(&status, cfg)
	if !status.StreamingUnavailable {
		t.Fatal("legacy generic selection lost its observation")
	}
}

func TestWhisperCPPStreamingCapabilityCannotBeRetried(t *testing.T) {
	cfg := config.Default()
	cfg.CompatibilityProfile = compatibility.WhisperCPP
	service := &Service{}
	var status FileTranscriptionStatus
	service.applyFileStreamingCapabilityLocked(&status, cfg)
	if !status.StreamingUnavailable || !status.StreamingProfileUnavailable {
		t.Fatal("native completed-only capability lost")
	}
	cfg.CompatibilityProfile = compatibility.VLLM
	service.applyFileStreamingCapabilityLocked(&status, cfg)
	if status.StreamingUnavailable || status.StreamingProfileUnavailable {
		t.Fatal("native restriction leaked to vLLM")
	}
}
