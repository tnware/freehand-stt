//go:build darwin

package platform

import (
	"github.com/tnware/freehand-stt/internal/hotkey"
	"testing"
)

func TestDarwinPhysicalKeyAdapter(t *testing.T) {
	var a keyboardAdapter
	for _, tc := range []struct {
		code uint16
		down bool
		key  uint32
		emit bool
	}{
		{59, true, 0x11, true}, {62, true, 0, false}, {59, false, 0, false}, {62, false, 0x11, true},
		{54, true, 0x5B, true}, {55, true, 0, false}, {54, false, 0, false}, {55, false, 0x5B, true},
		{0, true, 'A', true}, {0, true, 0, false}, {0, false, 'A', true}, {111, true, 0x7B, true}, {90, true, 0x83, true},
	} {
		key, down, ok := a.event(keyboardEvent{code: tc.code, down: tc.down})
		if key != tc.key || ok != tc.emit || (ok && down != tc.down) {
			t.Fatalf("%+v -> %x %v %v", tc, key, down, ok)
		}
	}
}

func TestDarwinAdapterModifierOnlyHold(t *testing.T) {
	var a keyboardAdapter
	r := hotkey.Reducer{Chord: hotkey.Chord{Modifiers: hotkey.Ctrl | hotkey.Alt}}
	for _, tc := range []struct {
		code uint16
		down bool
		edge hotkey.Edge
	}{{59, true, hotkey.NoEdge}, {61, true, hotkey.Pressed}, {62, true, hotkey.NoEdge}, {59, false, hotkey.NoEdge}, {62, false, hotkey.Released}} {
		key, down, ok := a.event(keyboardEvent{code: tc.code, down: tc.down})
		edge := hotkey.NoEdge
		if ok {
			edge = r.Event(key, down)
		}
		if edge != tc.edge {
			t.Fatalf("%+v = %v", tc, edge)
		}
	}
}
