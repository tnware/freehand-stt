package hotkey

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
)

type Modifier uint8

const (
	Ctrl Modifier = 1 << iota
	Alt
	Shift
	Meta
)

// Chord is a modifier set plus an optional primary key. A zero Key means a
// modifier-only chord, which only hold-to-talk accepts: RegisterHotKey takes a
// virtual-key code, so a toggle shortcut cannot be modifiers alone, while the
// hold hook sees raw modifier edges and can.
type Chord struct {
	Modifiers Modifier
	Key       uint32
}

// ModifierOnly reports whether this chord has no primary key.
func (c Chord) ModifierOnly() bool { return c.Key == 0 }

// ModifierCount is how many distinct modifiers the chord carries.
func (c Chord) ModifierCount() int {
	count := 0
	for _, bit := range []Modifier{Ctrl, Alt, Shift, Meta} {
		if c.Modifiers&bit != 0 {
			count++
		}
	}
	return count
}

// minModifierOnly is the floor for a modifier-only chord. A single modifier
// would arm hold-to-talk on almost every keystroke the user types.
const minModifierOnly = 2

var keyCodes = func() map[string]uint32 {
	m := map[string]uint32{"SPACE": 0x20}
	for r := 'A'; r <= 'Z'; r++ {
		m[string(r)] = uint32(r)
	}
	for r := '0'; r <= '9'; r++ {
		m[string(r)] = uint32(r)
	}
	for i := 1; i <= 24; i++ {

		m[fmt.Sprintf("F%d", i)] = uint32(0x6F + i)
	}
	return m
}()

// Parse accepts a toggle/show chord. Use ParseHold for a value that may be
// modifiers only.
func Parse(value string) (Chord, error) { return ParseFor(ToggleRecording, value) }

// ParseHold additionally accepts a modifier-only chord, for hold-to-talk.
func ParseHold(value string) (Chord, error) { return ParseFor(HoldToTalk, value) }

func ParseFor(action ShortcutAction, value string) (Chord, error) {
	return parseForPlatform(action, value, runtime.GOOS)
}

func parseForPlatform(action ShortcutAction, value, platform string) (Chord, error) {
	policy, ok := policyForPlatform(action, platform)
	if !ok {
		return Chord{}, reject(RejectionInvalidAction, action, "shortcut action is invalid")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		if !policy.Required {
			return Chord{}, nil
		}
		return Chord{}, reject(RejectionIncomplete, action, ActionLabel(action)+" requires a shortcut")
	}
	parts := strings.Split(value, "+")
	if chord, onlyModifiers := parseModifierOnlyForPlatform(parts, platform); onlyModifiers {
		if policy.ModifierOnlyMinimum == 0 {
			return Chord{}, reject(RejectionIncomplete, action, ActionLabel(action)+" needs a primary key; use "+Requirement(policy))
		}
		if chord.ModifierCount() < policy.ModifierOnlyMinimum {
			return Chord{}, reject(RejectionIncomplete, action, fmt.Sprintf("%s needs at least %d modifiers when no primary key is used", ActionLabel(action), policy.ModifierOnlyMinimum))
		}
		return chord, nil
	}
	var out Chord
	seenModifiers := Modifier(0)
	for i, raw := range parts {
		part := strings.ToUpper(strings.TrimSpace(raw))
		if part == "" {
			return Chord{}, reject(RejectionIncomplete, action, "shortcut contains an empty key")
		}
		last := i == len(parts)-1
		if !last {
			mod := modifierForPlatform(part, platform)
			if mod == 0 {
				return Chord{}, reject(RejectionUnsupported, action, fmt.Sprintf("%q is not a supported modifier; use Ctrl, Alt, Shift, or Win", raw))
			}
			if seenModifiers&mod != 0 {
				return Chord{}, reject(RejectionUnsupported, action, "shortcut contains the same modifier more than once")
			}
			seenModifiers |= mod
			out.Modifiers |= mod
			continue
		}
		key, ok := keyCodes[part]
		if part == "F12" && platform != "darwin" {
			return Chord{}, reject(RejectionReserved, action, "F12 is reserved by Windows and cannot be used; use "+Requirement(policy))
		}
		if !ok || !primaryForPlatform(key, platform) {
			return Chord{}, reject(RejectionUnsupported, action, fmt.Sprintf("%q is not supported; use %s", raw, Requirement(policy)))
		}
		out.Key = key
	}
	if out.Key == 0 {
		return Chord{}, reject(RejectionIncomplete, action, "shortcut needs a primary key")
	}
	if out.Modifiers == 0 && !dedicatedPrimary(out.Key) {
		return Chord{}, reject(RejectionIncomplete, action, ActionLabel(action)+" needs a modifier with that key; only "+strings.Join(policy.DedicatedPrimaryGroups, ", ")+" may be used on their own")
	}
	return out, nil
}

// parseModifierOnly succeeds only when every part is a modifier.
func parseModifierOnlyForPlatform(parts []string, platform string) (Chord, bool) {
	var out Chord
	for _, raw := range parts {
		part := strings.ToUpper(strings.TrimSpace(raw))
		if part == "" {
			return Chord{}, false
		}
		mod := modifierForPlatform(part, platform)
		if mod == 0 || out.Modifiers&mod != 0 {
			return Chord{}, false
		}
		out.Modifiers |= mod
	}
	return out, true
}

func modifierForPlatform(part, platform string) Modifier {
	switch part {
	case "CMDORCTRL":
		if platform == "darwin" {
			return Meta
		}
		return Ctrl
	case "CTRL", "CONTROL":
		return Ctrl
	case "ALT", "OPTION", "OPTIONORALT":
		return Alt
	case "SHIFT":
		return Shift
	case "META", "CMD", "COMMAND", "WIN", "SUPER":
		return Meta
	default:
		return 0
	}
}

func (c Chord) String() string {
	parts := make([]string, 0, 5)
	for _, item := range []struct {
		bit  Modifier
		name string
	}{{Ctrl, "Ctrl"}, {Alt, "Alt"}, {Shift, "Shift"}, {Meta, "Super"}} {
		if c.Modifiers&item.bit != 0 {
			parts = append(parts, item.name)
		}
	}
	if c.ModifierOnly() {
		if len(parts) == 0 {
			return ""
		}
		return strings.Join(parts, "+")
	}
	keys := make([]string, 0, len(keyCodes))
	for name := range keyCodes {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if keyCodes[name] == c.Key {
			if name == "SPACE" {
				name = "Space"
			}
			return strings.Join(append(parts, name), "+")
		}
	}
	return ""
}

func dedicatedPrimary(vk uint32) bool { return vk >= 0x7C && vk <= 0x87 }

func NormalizeFor(action ShortcutAction, value string) (string, error) {
	chord, err := ParseFor(action, value)
	if err != nil {
		return "", err
	}
	return chord.String(), nil
}

func assignmentValue(assignments ShortcutAssignments, action ShortcutAction) string {
	switch action {
	case ToggleRecording:
		return assignments.ToggleRecording
	case ShowFreehand:
		return assignments.ShowFreehand
	case HoldToTalk:
		return assignments.HoldToTalk
	default:
		return ""
	}
}

func AssignmentMatches(action ShortcutAction, candidate Chord, assignments ShortcutAssignments) bool {
	value := assignmentValue(assignments, action)
	if strings.TrimSpace(value) == "" {
		return false
	}
	chord, err := ParseFor(action, value)
	return err == nil && chord == candidate
}

// FindConflict compares parsed chords, so aliases and modifier ordering cannot
// evade duplicate detection.
func FindConflict(action ShortcutAction, candidate Chord, assignments ShortcutAssignments) (ShortcutAction, bool) {
	for _, other := range []ShortcutAction{ToggleRecording, ShowFreehand, HoldToTalk} {
		if other == action {
			continue
		}
		value := assignmentValue(assignments, other)
		if strings.TrimSpace(value) == "" {
			continue
		}
		chord, err := ParseFor(other, value)
		if err == nil && chord == candidate {
			return other, true
		}
	}
	return "", false
}

// ValidateAssignments enforces required bindings, every action-specific chord
// form, and cross-action uniqueness from one policy.
func ValidateAssignments(assignments ShortcutAssignments) error {
	seen := make(map[Chord]ShortcutAction)
	for _, action := range []ShortcutAction{ToggleRecording, ShowFreehand, HoldToTalk} {
		value := assignmentValue(assignments, action)
		policy, _ := PolicyFor(action)
		if strings.TrimSpace(value) == "" && !policy.Required {
			continue
		}
		chord, err := ParseFor(action, value)
		if err != nil {
			return err
		}
		if prior, exists := seen[chord]; exists {
			return &Rejection{
				Kind:              RejectionDuplicate,
				Action:            action,
				ConflictingAction: prior,
				message:           fmt.Sprintf("%s and %s must use different shortcuts", ActionLabel(prior), ActionLabel(action)),
			}
		}
		seen[chord] = action
	}
	return nil
}
