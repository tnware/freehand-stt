package platform

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCaptionKeepsUnicodeTailInOneCachedRow(t *testing.T) {
	text := BoundedOverlayCaption(strings.Repeat("old ", 100) + "最新\nwords\tfinish\u2028here")
	if utf8.RuneCountInString(text) > 180 || strings.ContainsAny(text, "\n\t\u2028") {
		t.Fatal("caption was not bounded to a single row")
	}
	var c overlayCaptionCache
	calls := 0
	measure := func(s string) float64 { calls++; return float64(utf8.RuneCountInString(s)) }
	got := c.fit(text, 12, 1, measure)
	if got != "finish here" {
		t.Fatalf("tail = %q", got)
	}
	measured := calls
	if c.fit(text, 12, 1, measure) != got || calls != measured {
		t.Fatal("unchanged caption was measured again")
	}
	_ = c.fit(text, 20, 1, measure)
	if calls == measured {
		t.Fatal("resized caption retained stale fit")
	}
}
