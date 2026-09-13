package savedconnection

import "testing"

func TestLimitUsesNormalPerPurposeBound(t *testing.T) {
	for _, purpose := range []Purpose{Voice, Transcription, Cleanup, Speech} {
		t.Run(string(purpose), func(t *testing.T) {
			if got := Limit(purpose); got != MaxPerPurpose {
				t.Fatalf("Limit(%q) = %d, want %d", purpose, got, MaxPerPurpose)
			}
		})
	}
}
