package inference

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestResponseMetadataSanitizerCoversEveryString(t *testing.T) {
	const key = "metadata-field-canary"
	var metadata ResponseMetadata
	// Automatically seed new nested string fields too: adding a retained string
	// to the DTO must extend the publication sanitizer, not silently bypass it.
	var seed func(reflect.Value)
	seed = func(value reflect.Value) {
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				seed(value.Field(i))
			}
		case reflect.String:
			value.SetString("prefix-" + key)
		case reflect.Slice:
			value.Set(reflect.MakeSlice(value.Type(), 1, 1))
			seed(value.Index(0))
		case reflect.Pointer:
			value.Set(reflect.New(value.Type().Elem()))
			seed(value.Elem())
		}
	}
	seed(reflect.ValueOf(&metadata).Elem())
	metadata.UsageReportCount = 1
	clean := sanitizeResponseMetadata(metadata, key)
	body, err := json.Marshal(clean)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), key) {
		t.Fatal("a retained string bypassed the shared sanitizer")
	}
	if metadata.Provider != "prefix-"+key || metadata.DetectedLanguages[0] != "prefix-"+key {
		t.Fatal("sanitizer mutated its input")
	}
	if got := sanitizeResponseMetadata(ResponseMetadata{Usage: Usage{Type: key}, UsageReportCount: 1}, key); got.Usage.Reported() || got.UsageReportCount != 0 {
		t.Fatal("removing the only usage field retained false report coverage")
	}
}

func TestTranscriptionSplitCredentialStillRejectsCompletingDelta(t *testing.T) {
	const key = "split-key-canary"
	stream := "data: {\"type\":\"transcript.text.delta\",\"delta\":\"split-key-\"}\n\n" +
		"data: {\"type\":\"transcript.text.delta\",\"delta\":\"canary\"}\n\n"
	var deltas strings.Builder
	result, err := readTranscriptionSSE(strings.NewReader(stream), key, func(s string) { deltas.WriteString(s) })
	failure, ok := err.(*Error)
	if !ok || failure.Kind != "credential_reflection" || result.Text != "" {
		t.Fatalf("expected split credential rejection, got %v", err)
	}
	// Existing streaming semantics may publish a prefix, but never the delta
	// completing the literal key. This change does not redesign text buffering.
	if deltas.String() != "split-key-" || strings.Contains(deltas.String(), key) {
		t.Fatal("completing credential delta reached the callback")
	}
}
