package buildinfo

import "testing"

func TestCurrentReturnsBoundedImmutableMetadata(t *testing.T) {
	service := NewService("Freehand", "0.1.0-alpha.1", "0.1.0.1", true)
	first := service.Current()
	second := service.Current()
	if first != second {
		t.Fatalf("build identity changed: %#v then %#v", first, second)
	}
	if first.ProductName != "Freehand" || first.Version != "0.1.0-alpha.1" {
		t.Fatalf("release identity = %#v", first)
	}
	if first.WindowsVersion != "0.1.0.1" || !first.Development {
		t.Fatalf("build classification = %#v", first)
	}
	if first.GoVersion == "" || first.WailsVersion == "" {
		t.Fatalf("runtime metadata is incomplete: %#v", first)
	}
}
