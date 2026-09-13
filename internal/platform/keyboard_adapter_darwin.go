//go:build darwin

package platform

// Chord IDs are the existing wire convention, not macOS virtual keycodes.
// Like pinned Wails Carbon shortcuts these are physical ANSI/QWERTY positions.
var keyboardWireKeys = map[uint16]uint32{
	0: 'A', 1: 'S', 2: 'D', 3: 'F', 4: 'H', 5: 'G', 6: 'Z', 7: 'X', 8: 'C', 9: 'V', 11: 'B', 12: 'Q', 13: 'W', 14: 'E', 15: 'R', 16: 'Y', 17: 'T', 31: 'O', 32: 'U', 34: 'I', 35: 'P', 37: 'L', 38: 'J', 40: 'K', 45: 'N', 46: 'M',
	18: '1', 19: '2', 20: '3', 21: '4', 22: '6', 23: '5', 25: '9', 26: '7', 28: '8', 29: '0', 49: 0x20, 53: 0x1B,
	122: 0x70, 120: 0x71, 99: 0x72, 118: 0x73, 96: 0x74, 97: 0x75, 98: 0x76, 100: 0x77, 101: 0x78, 109: 0x79, 103: 0x7A, 111: 0x7B, 105: 0x7C, 107: 0x7D, 113: 0x7E, 106: 0x7F, 64: 0x80, 79: 0x81, 80: 0x82, 90: 0x83,
	59: 0x11, 62: 0x11, 56: 0x10, 60: 0x10, 58: 0x12, 61: 0x12, 55: 0x5B, 54: 0x5B,
}

type keyboardEvent struct {
	code       uint16
	down, lost bool
}
type keyboardAdapter struct{ held [128]bool }

// Aggregate the two physical sides before the shared reducer sees a modifier.
// A second side down / first side up must not change its logical modifier set.
func (a *keyboardAdapter) event(e keyboardEvent) (uint32, bool, bool) {
	if e.code >= 128 || a.held[e.code] == e.down {
		return 0, false, false
	}
	a.held[e.code] = e.down
	key, ok := keyboardWireKeys[e.code]
	if !ok {
		return 0xffff, e.down, true
	} // unsupported primary, never typed text
	if key == 0x11 || key == 0x10 || key == 0x12 || key == 0x5B {
		for code, other := range keyboardWireKeys {
			if code != e.code && other == key && a.held[code] {
				return 0, false, false
			}
		}
	}
	return key, e.down, true
}
