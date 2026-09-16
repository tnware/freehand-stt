package hotkey

import (
	"runtime"
	"strings"
)

type CaptureState uint8

const (
	CaptureWaiting CaptureState = iota
	CaptureComplete
	CaptureCanceled
	CaptureRejected
)

type CaptureResult struct {
	State CaptureState
	Chord Chord
	Err   error
}

// CaptureReducer turns physical keyboard edges into one recorded chord.
type CaptureReducer struct {
	Policy ShortcutPolicy

	modifiers Modifier
	held      Modifier
	primary   uint32
	candidate Chord
	settled   bool
}

func (r *CaptureReducer) effectivePolicy() ShortcutPolicy {
	if policy, ok := PolicyFor(r.Policy.Action); ok {
		return policy
	}
	policy, _ := PolicyFor(ToggleRecording)
	return policy
}

// Preview is the normalized chord currently held during capture. It is safe
// to publish because it contains only bounded key names, never typed text.
func (r *CaptureReducer) Preview() Chord {
	if r.primary != 0 {
		return r.candidate
	}
	return Chord{Modifiers: r.modifiers}
}

// Event consumes one physical keyboard edge while shortcut capture is active.
// A complete chord is returned only after the primary key is released so the
// native hook can suppress the full key press from existing global shortcuts.
func (r *CaptureReducer) Event(vk uint32, down bool) CaptureResult {
	policy := r.effectivePolicy()
	// A capture reports exactly one outcome. The keys still held when a chord
	// completes are released afterwards, and those edges must stay silent.
	if r.settled {
		return CaptureResult{State: CaptureWaiting}
	}
	if vk == 0x1B { // Escape always cancels capture.
		if down {
			return CaptureResult{State: CaptureCanceled}
		}
		return CaptureResult{State: CaptureWaiting}
	}
	if mod := modifierForVK(vk); mod != 0 {
		if down {
			r.modifiers |= mod
			r.held |= mod
			return CaptureResult{State: CaptureWaiting}
		}
		r.modifiers &^= mod
		// A primary key is already staged; its release completes the chord.
		if r.primary != 0 {
			return CaptureResult{State: CaptureWaiting}
		}
		if policy.ModifierOnlyMinimum > 0 {
			held := r.held
			if countModifiers(held) >= policy.ModifierOnlyMinimum {
				r.reset()
				return CaptureResult{State: CaptureComplete, Chord: Chord{Modifiers: held}}
			}
		}
		// Everything is released and nothing was recorded: say so now rather
		// than leaving the recorder waiting until it times out.
		if r.modifiers == 0 {
			r.reset()
			if policy.ModifierOnlyMinimum > 0 {
				return CaptureResult{State: CaptureRejected, Err: reject(RejectionIncomplete, policy.Action, "Hold to talk needs two modifiers, a modifier plus a supported key, or F13-F24 on its own")}
			}
			return CaptureResult{State: CaptureRejected, Err: reject(RejectionIncomplete, policy.Action, "Add a primary key; use "+Requirement(policy))}
		}
		return CaptureResult{State: CaptureWaiting}
	}
	if !down {
		if vk == r.primary && r.primary != 0 {
			chord := r.candidate
			r.reset()
			return CaptureResult{State: CaptureComplete, Chord: chord}
		}
		return CaptureResult{State: CaptureWaiting}
	}
	if r.primary != 0 {
		return CaptureResult{State: CaptureWaiting}
	}
	if vk == 0x7B && runtime.GOOS != "darwin" { // Windows reserves F12.
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionReserved, policy.Action, "F12 is reserved by Windows and cannot be used; use "+Requirement(policy))}
	}
	if !supportedPrimary(vk) || !primaryForPlatform(vk, runtime.GOOS) {
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionUnsupported, policy.Action, "That key is not supported; use "+Requirement(policy))}
	}
	if r.modifiers == 0 && !dedicatedPrimary(vk) {
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionIncomplete, policy.Action, "Add Ctrl, Alt, Shift, or Win; only "+strings.Join(policy.DedicatedPrimaryGroups, ", ")+" may be used on their own")}
	}
	r.primary = vk
	r.candidate = Chord{Modifiers: r.modifiers, Key: vk}
	return CaptureResult{State: CaptureWaiting}
}

// reset clears the staged chord so one reducer can serve a retry.
func (r *CaptureReducer) reset() {
	r.primary = 0
	r.candidate = Chord{}
	r.held = 0
	r.settled = true
}

func countModifiers(m Modifier) int {
	return Chord{Modifiers: m}.ModifierCount()
}

func supportedPrimary(vk uint32) bool {
	for _, candidate := range keyCodes {
		if candidate == vk {
			return true
		}
	}
	return false
}
