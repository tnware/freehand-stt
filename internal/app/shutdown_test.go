package app

import "testing"

type countingNativeInput struct{ closed int }

func (i *countingNativeInput) Close() error { i.closed++; return nil }

func TestNativeTargetOwnerClosesExactlyOnce(t *testing.T) {
	input := &countingNativeInput{}
	a := &App{nativeInput: input}
	a.closeResources()
	a.closeResources()
	if input.closed != 1 {
		t.Fatalf("native retained targets closed %d times", input.closed)
	}
}
