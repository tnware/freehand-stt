package hotkey

import (
	"errors"
	"fmt"
	"runtime"
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
	return policiesForPlatform(runtime.GOOS)
}

func policiesForPlatform(platform string) []ShortcutPolicy {
	policies := []ShortcutPolicy{
		policy(ToggleRecording, false, 0, "Ctrl+Shift+Space", "CmdOrCtrl+Shift+Space"),
		policy(ShowFreehand, false, 0, ""),
		policy(HoldToTalk, false, minModifierOnly, ""),
	}
	if platform == "darwin" {
		for i := range policies {
			policies[i].ModifiedPrimaryGroups = []string{"A-Z", "0-9", "Space", "F1-F20"}
			policies[i].DedicatedPrimaryGroups = []string{"F13-F20"}
		}
		policies[0].DefaultShortcut = "Shift+Super+Space"
	}
	return policies
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
	return policyForPlatform(action, runtime.GOOS)
}

func policyForPlatform(action ShortcutAction, platform string) (ShortcutPolicy, bool) {
	for _, item := range policiesForPlatform(platform) {
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
