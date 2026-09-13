package credential

import "testing"

func TestKeyringRequiresExplicitAccountBeforeNativeAccess(t *testing.T) {
	k := Keyring{}
	if _, err := k.Get(); err == nil || err.Error() != "credential account is required" {
		t.Fatalf("Get without account = %v", err)
	}
	if err := k.Set("fixture-only"); err == nil || err.Error() != "credential account is required" {
		t.Fatalf("Set without account = %v", err)
	}
	if err := k.Delete(); err == nil || err.Error() != "credential account is required" {
		t.Fatalf("Delete without account = %v", err)
	}
	if k.Configured() {
		t.Fatal("empty account is configured")
	}
}
