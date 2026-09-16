package hotkey

type Edge uint8

const (
	NoEdge Edge = iota
	Pressed
	Released
)

type Reducer struct {
	Chord   Chord
	mods    Modifier
	primary bool
	active  bool
}

func (r *Reducer) Event(vk uint32, down bool) Edge {
	if r.Chord == (Chord{}) {
		return NoEdge
	}
	if mod := modifierForVK(vk); mod != 0 {
		if down {
			r.mods |= mod
			// A modifier-only chord has no primary key, so completing the
			// modifier set is itself the press edge.
			if r.Chord.ModifierOnly() && !r.active && r.mods&r.Chord.Modifiers == r.Chord.Modifiers {
				r.active = true
				return Pressed
			}
			return NoEdge
		}
		r.mods &^= mod
		if r.active && r.mods&r.Chord.Modifiers != r.Chord.Modifiers {
			r.active = false
			return Released
		}
		return NoEdge
	}
	if r.Chord.ModifierOnly() {
		// Ordinary typing must not disturb a modifier-only hold.
		return NoEdge
	}
	if vk != r.Chord.Key {
		return NoEdge
	}
	if down {
		if r.primary {
			return NoEdge
		}
		r.primary = true
		if r.mods&r.Chord.Modifiers == r.Chord.Modifiers {
			r.active = true
			return Pressed
		}
		return NoEdge
	}
	r.primary = false
	if r.active {
		r.active = false
		return Released
	}
	return NoEdge
}

func (r *Reducer) ForceRelease() Edge {
	r.primary = false
	r.mods = 0
	if r.active {
		r.active = false
		return Released
	}
	return NoEdge
}

func modifierForVK(vk uint32) Modifier {
	switch vk {
	case 0x10, 0xA0, 0xA1:
		return Shift
	case 0x11, 0xA2, 0xA3:
		return Ctrl
	case 0x12, 0xA4, 0xA5:
		return Alt
	case 0x5B, 0x5C:
		return Meta
	default:
		return 0
	}
}
