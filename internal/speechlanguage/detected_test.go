package speechlanguage

import (
	"reflect"
	"testing"
)

func TestDetectedLanguagesKeepEarlierTurnsAndBoundOverflow(t *testing.T) {
	first := []string{"fr-FR"}
	got := MergeDetected(first, []string{"auto", "en-US", "fr-FR", "in\nvalid"})
	if !reflect.DeepEqual(first, []string{"fr-FR"}) || !reflect.DeepEqual(got, []string{"fr-FR", "en-US"}) {
		t.Fatalf("language evidence lost or mutated: %v", got)
	}
	got = MergeDetected(got, []string{"de-DE", "it-IT", "es-ES", "pt-PT", "nl-NL", "ja-JP", "ko-KR"})
	if len(got) != 8 || got[7] != "mul" || English(got[7]) {
		t.Fatalf("overflow was not bounded multilingual evidence: %v", got)
	}
}
