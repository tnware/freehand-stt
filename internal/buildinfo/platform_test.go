package buildinfo

import (
	"encoding/json"
	"runtime"
	"testing"
)

func TestCurrentReportsNativePlatform(t *testing.T) {
	b, _ := json.Marshal(NewService("Freehand", "test", "0", true).Current())
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["platform"] != runtime.GOOS {
		t.Fatalf("platform=%v want %s", fields["platform"], runtime.GOOS)
	}
}
