package hotkey

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ShortcutAction identifies one bounded product action. It is deliberately
// not an arbitrary string: future profile shortcuts need their own policy
// instead of silently inheriting one of these behaviors.
type ShortcutAction string

const (
	ToggleRecording ShortcutAction = "toggle"
	ShowFreehand    ShortcutAction = "show"
	HoldToTalk      ShortcutAction = "hold"
)

type ShortcutRejectionKind string

const (
	RejectionInvalidAction ShortcutRejectionKind = "invalid-action"
	RejectionIncomplete    ShortcutRejectionKind = "incomplete"
	RejectionUnsupported   ShortcutRejectionKind = "unsupported"
	RejectionReserved      ShortcutRejectionKind = "reserved"
	RejectionDuplicate     ShortcutRejectionKind = "duplicate"
	RejectionUnavailable   ShortcutRejectionKind = "unavailable"
	RejectionTimedOut      ShortcutRejectionKind = "timed-out"
)

// ShortcutPolicy is renderer-safe metadata for the exact native policy. The
// compact key groups keep the binding bounded while still letting settings
// explain every supported chord form before capture starts.
type ShortcutPolicy struct {
	Action                    ShortcutAction `json:"action"`
	Required                  bool           `json:"required"`
	ModifiedPrimaryGroups     []string       `json:"modifiedPrimaryGroups"`
	DedicatedPrimaryGroups    []string       `json:"dedicatedPrimaryGroups"`
	ModifierOnlyMinimum       int            `json:"modifierOnlyMinimum"`
	DefaultShortcut           string         `json:"defaultShortcut,omitempty"`
	DefaultAliases            []string       `json:"defaultAliases,omitempty"`
	ExternalAvailabilityKnown bool           `json:"externalAvailabilityKnown"`
}

// ShortcutAssignments is the complete, bounded set of editable shortcuts.
// Capture receives the current draft so duplicate validation stays in Go even
// before the settings transaction is saved.
type ShortcutAssignments struct {
	ToggleRecording string `json:"toggleRecording"`
	ShowFreehand    string `json:"showFreehand"`
	HoldToTalk      string `json:"holdToTalk,omitempty"`
}

func Policies() []ShortcutPolicy {
	return []ShortcutPolicy{
		policy(ToggleRecording, true, 0, "Ctrl+Shift+Space", "CmdOrCtrl+Shift+Space"),
		policy(ShowFreehand, true, 0, "Ctrl+Shift+D", "CmdOrCtrl+Shift+D"),
		policy(HoldToTalk, false, minModifierOnly, ""),
	}
}

func policy(action ShortcutAction, required bool, modifierOnlyMinimum int, defaultShortcut string, aliases ...string) ShortcutPolicy {
	return ShortcutPolicy{
		Action:                    action,
		Required:                  required,
		ModifiedPrimaryGroups:     []string{"A-Z", "0-9", "Space", "F1-F11", "F13-F24"},
		DedicatedPrimaryGroups:    []string{"F13-F24"},
		ModifierOnlyMinimum:       modifierOnlyMinimum,
		DefaultShortcut:           defaultShortcut,
		DefaultAliases:            aliases,
		ExternalAvailabilityKnown: false,
	}
}

func PolicyFor(action ShortcutAction) (ShortcutPolicy, bool) {
	for _, item := range Policies() {
		if item.Action == action {
			return item, true
		}
	}
	return ShortcutPolicy{}, false
}

func ActionLabel(action ShortcutAction) string {
	switch action {
	case ToggleRecording:
		return "Toggle recording"
	case ShowFreehand:
		return "Show Freehand"
	case HoldToTalk:
		return "Hold to talk"
	default:
		return "Shortcut"
	}
}

// Rejection is an expected, renderer-safe shortcut validation outcome. Native
// hook and lifecycle failures remain ordinary errors.
type Rejection struct {
	Kind              ShortcutRejectionKind
	Action            ShortcutAction
	ConflictingAction ShortcutAction
	message           string
}

func (e *Rejection) Error() string { return e.message }

func reject(kind ShortcutRejectionKind, action ShortcutAction, message string) error {
	return &Rejection{Kind: kind, Action: action, message: message}
}

func NewRejection(kind ShortcutRejectionKind, action ShortcutAction, message string) error {
	return reject(kind, action, message)
}

func RejectionDetails(err error) (*Rejection, bool) {
	var rejection *Rejection
	ok := errors.As(err, &rejection)
	return rejection, ok
}

func Requirement(policy ShortcutPolicy) string {
	modified := "a modifier plus " + strings.Join(policy.ModifiedPrimaryGroups, ", ")
	forms := []string{modified, strings.Join(policy.DedicatedPrimaryGroups, ", ") + " on its own"}
	if policy.ModifierOnlyMinimum > 0 {
		forms = append(forms, fmt.Sprintf("%d modifiers on their own", policy.ModifierOnlyMinimum))
	}
	return strings.Join(forms, "; or ")
}

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
		if i == 12 {
			continue
		}
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
	policy, ok := PolicyFor(action)
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
	if chord, onlyModifiers := parseModifierOnly(parts); onlyModifiers {
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
			mod := modifierForName(part)
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
		if !ok {
			if part == "F12" {
				return Chord{}, reject(RejectionReserved, action, "F12 is reserved by Windows and cannot be used; use "+Requirement(policy))
			}
			return Chord{}, reject(RejectionUnsupported, action, fmt.Sprintf("%q is not supported; use %s", raw, Requirement(policy)))
		}
		out.Key = key
	}
	if out.Key == 0 {
		return Chord{}, reject(RejectionIncomplete, action, "shortcut needs a primary key")
	}
	if out.Modifiers == 0 && !dedicatedPrimary(out.Key) {
		return Chord{}, reject(RejectionIncomplete, action, ActionLabel(action)+" needs a modifier with that key; only F13-F24 may be used on their own")
	}
	return out, nil
}

// parseModifierOnly succeeds only when every part is a modifier.
func parseModifierOnly(parts []string) (Chord, bool) {
	var out Chord
	for _, raw := range parts {
		part := strings.ToUpper(strings.TrimSpace(raw))
		if part == "" {
			return Chord{}, false
		}
		mod := modifierForName(part)
		if mod == 0 || out.Modifiers&mod != 0 {
			return Chord{}, false
		}
		out.Modifiers |= mod
	}
	return out, true
}

func modifierForName(part string) Modifier {
	switch part {
	case "CTRL", "CONTROL", "CMDORCTRL":
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
	if vk == 0x7B { // F12 is reserved by Windows.
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionReserved, policy.Action, "F12 is reserved by Windows and cannot be used; use "+Requirement(policy))}
	}
	if !supportedPrimary(vk) {
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionUnsupported, policy.Action, "That key is not supported; use "+Requirement(policy))}
	}
	if r.modifiers == 0 && !dedicatedPrimary(vk) {
		return CaptureResult{State: CaptureRejected, Err: reject(RejectionIncomplete, policy.Action, "Add Ctrl, Alt, Shift, or Win; only F13-F24 may be used on their own")}
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
