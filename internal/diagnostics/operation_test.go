package diagnostics

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestOperationCorrelationIsBoundedAndLocalToContext(t *testing.T) {
	var logs bytes.Buffer
	base := slog.New(slog.NewTextHandler(&logs, nil)).With("component", "postprocess")
	root, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, tc := range []struct {
		workflow   Workflow
		generation uint64
		want       string
	}{
		{Dictation, 7, "workflow=dictation generation=7"},
		{File, 7, "workflow=file generation=7"},
		{Workflow("private user label"), 7, ""},
		{Dictation, 0, ""},
	} {
		logs.Reset()
		ctx := WithOperation(root, tc.workflow, tc.generation)
		OperationLogger(ctx, base).Info("stage completed")
		output := logs.String()
		if !strings.Contains(output, "component=postprocess") {
			t.Fatal("lost component owner")
		}
		if tc.want != "" && !strings.Contains(output, tc.want) {
			t.Fatalf("missing correlation: %s", output)
		}
		if tc.want == "" && strings.Contains(output, "workflow=") {
			t.Fatal("invalid correlation admitted")
		}
	}
	logs.Reset()
	OperationLogger(root, base).Info("unrelated operation")
	if strings.Contains(logs.String(), "generation=") {
		t.Fatal("child correlation mutated parent logger or context")
	}
	ctx := WithOperation(root, Dictation, 8)
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatal("correlation lost parent cancellation")
	}
	if OperationLogger(ctx, nil) == nil {
		t.Fatal("missing logger must use non-emitting fallback")
	}
}
