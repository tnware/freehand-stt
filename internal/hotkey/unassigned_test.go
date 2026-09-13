package hotkey

import "testing"

func TestUnassignedHoldNeverEmitsAnEdge(t *testing.T) {
	reducer := Reducer{}
	for _, key := range []uint32{0x11, 0x10, 0x12, 0x5B, 0x41, 0x20} {
		for _, down := range []bool{true, false} {
			if edge := reducer.Event(key, down); edge != NoEdge {
				t.Fatalf("unassigned hold emitted %v for key %x down=%v", edge, key, down)
			}
		}
	}
}
