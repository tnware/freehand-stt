package releaseinfo

import "testing"

func TestPrereleaseNumericIdentifiersAreCanonical(t *testing.T) {
	for _, version := range []string{"0.1.0-alpha.01", "0.1.0-beta.001", "0.1.0-01.1", "0.1.0-alpha.00.1", "0.1.0-alpha.00000000000000000000000000001.1"} {
		t.Run(version, func(t *testing.T) {
			if got, err := WindowsVersion(version); err == nil {
				t.Errorf("WindowsVersion(%q) normalized invalid source to %q", version, got)
			}
			if _, err := Parse([]byte("info: {productName: Freehand, productIdentifier: io.github.tnware.freehand, version: " + version + "}")); err == nil {
				t.Errorf("Parse accepted invalid source %q", version)
			}
		})
	}
}

func TestParseUsesTheWailsReleaseSource(t *testing.T) {
	info, err := Parse([]byte(`
info:
  productName: "Freehand"
  productIdentifier: "io.github.tnware.freehand"
  version: "0.1.0-alpha.7"
`))
	if err != nil {
		t.Fatal(err)
	}
	if info.ProductName != "Freehand" || info.ProductIdentifier != "io.github.tnware.freehand" {
		t.Fatalf("identity = %#v", info)
	}
	if info.Version != "0.1.0-alpha.7" || info.WindowsVersion != "0.1.0.7" {
		t.Fatalf("versions = display %q Windows %q", info.Version, info.WindowsVersion)
	}
}

func TestWindowsVersionDerivation(t *testing.T) {
	for _, test := range []struct {
		version string
		want    string
		valid   bool
	}{
		{version: "1.2.3", want: "1.2.3.0", valid: true},
		{version: "0.1.0-alpha.1", want: "0.1.0.1", valid: true},
		{version: "2.0.0-rc.42+build.9", want: "2.0.0.42", valid: true},
		{version: "0.1.0-01alpha.1+build.001", want: "0.1.0.1", valid: true},
		{version: "0.1.0-0-1.1", want: "0.1.0.1", valid: true},
		{version: "0.1.0-alpha.0.1", want: "0.1.0.1", valid: true},
		{version: "1.2.3+001", want: "1.2.3.0", valid: true},
		{version: "65535.65535.65535-alpha.65535", want: "65535.65535.65535.65535", valid: true},
		{version: "0.1.0-alpha.65536", valid: false},
		{version: "1.2", valid: false},
		{version: "1.2.3-alpha", valid: false},
		{version: "1.2.3-alpha.0", valid: false},
		{version: "65536.0.0", valid: false},
	} {
		got, err := WindowsVersion(test.version)
		if test.valid && (err != nil || got != test.want) {
			t.Fatalf("WindowsVersion(%q) = %q, %v; want %q", test.version, got, err, test.want)
		}
		if !test.valid && err == nil {
			t.Fatalf("WindowsVersion(%q) unexpectedly accepted as %q", test.version, got)
		}
	}
}

func TestParseRejectsIncompleteIdentity(t *testing.T) {
	for _, data := range []string{
		`info: {productIdentifier: "io.github.tnware.freehand", version: "1.0.0"}`,
		`info: {productName: "Freehand", version: "1.0.0"}`,
		`info: {productName: "Freehand", productIdentifier: "io.github.tnware.freehand", version: "alpha"}`,
	} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatalf("invalid release identity was accepted: %s", data)
		}
	}
}
