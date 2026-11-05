package hotkey

import "testing"

func TestLinuxRequiresPrimaryForHold(t *testing.T) {
	policy, _ := policyForPlatform(HoldToTalk, "linux")
	if policy.ModifierOnlyMinimum != 0 {
		t.Fatal("Linux advertised modifier-only holds")
	}
	if _, err := parseForPlatform(HoldToTalk, "Ctrl+Shift", "linux"); err == nil {
		t.Fatal("Linux accepted modifier-only hold")
	}
	if _, err := parseForPlatform(HoldToTalk, "Ctrl+Shift+Space", "linux"); err != nil {
		t.Fatal(err)
	}
	for _, platform := range []string{"windows", "darwin"} {
		if _, err := parseForPlatform(HoldToTalk, "Ctrl+Shift", platform); err != nil {
			t.Fatalf("%s regression: %v", platform, err)
		}
	}
}
