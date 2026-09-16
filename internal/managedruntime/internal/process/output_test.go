package process

import (
	"strings"
	"testing"
)

func TestObserverPreservesParserPrefixAndReceivesEntireWrite(t *testing.T) {
	prefix := &boundedOutput{}
	var output []byte
	writer := observedOutput{prefix: prefix, output: func(stream string, p []byte) {
		if stream != "stdout" {
			t.Fatal("wrong stream")
		}
		output = append(output, p...)
	}, stream: "stdout"}
	input := "\x1b[31m" + strings.Repeat("a", outputLimit) + "tail\x1b[0m"
	n, err := writer.Write([]byte(input))
	if n != len(input) || err != nil || !prefix.overflow || string(prefix.bytes()) != input[:outputLimit] {
		t.Fatal("parser prefix/overflow changed")
	}
	if string(output) != input {
		t.Fatal("observer lost output beyond diagnostic prefix")
	}
}
