//go:build windows

package platform

import "testing"

func TestGDIPlusCanRestartAfterOverlayShutdown(t *testing.T) {
	if err := gdiplusInit(); err != nil {
		t.Fatal(err)
	}
	if gdiplusToken == 0 {
		t.Fatal("first GDI+ startup returned no token")
	}
	gdiplusRelease()
	if gdiplusToken != 0 {
		t.Fatal("GDI+ shutdown retained its token")
	}

	if err := gdiplusInit(); err != nil {
		t.Fatal(err)
	}
	if gdiplusToken == 0 {
		t.Fatal("GDI+ did not restart after overlay shutdown")
	}
	t.Cleanup(gdiplusRelease)
}
