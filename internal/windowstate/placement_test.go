package windowstate

import (
	"path/filepath"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestResolveRestoresRelativePlacementOnMatchingScreen(t *testing.T) {
	screen := testScreen("secondary-id", "secondary", false, application.Rect{X: -1600, Y: 40, Width: 1600, Height: 860})
	saved := Placement{ScreenID: screen.ID, ScreenName: screen.Name, X: 120, Y: 80, Width: 900, Height: 650}

	bounds, resolved, ok := Resolve(saved, []*application.Screen{screen}, 1080, 720, 560, 560)
	if !ok || resolved != screen {
		t.Fatalf("resolved screen = %#v, ok=%v", resolved, ok)
	}
	want := application.Rect{X: -1480, Y: 120, Width: 900, Height: 650}
	if bounds != want {
		t.Fatalf("bounds = %#v, want %#v", bounds, want)
	}
}

func TestResolveUsesScreenNameWhenWindowsIDChanged(t *testing.T) {
	screen := testScreen("new-handle", "same-device", false, application.Rect{X: 1920, Y: 0, Width: 1920, Height: 1040})
	saved := Placement{ScreenID: "old-handle", ScreenName: screen.Name, X: 50, Y: 60, Width: 800, Height: 600}

	bounds, resolved, ok := Resolve(saved, []*application.Screen{screen}, 1080, 720, 560, 560)
	if !ok || resolved != screen {
		t.Fatalf("resolved screen = %#v, ok=%v", resolved, ok)
	}
	if bounds.X != 1970 || bounds.Y != 60 {
		t.Fatalf("bounds = %#v", bounds)
	}
}

func TestResolveCentersOnPrimaryWhenSavedScreenDisappeared(t *testing.T) {
	primary := testScreen("primary", "primary", true, application.Rect{X: 0, Y: 0, Width: 1920, Height: 1040})
	saved := Placement{ScreenID: "missing", ScreenName: "missing", X: -5000, Y: 5000, Width: 1000, Height: 700}

	bounds, resolved, ok := Resolve(saved, []*application.Screen{primary}, 1080, 720, 560, 560)
	if !ok || resolved != primary {
		t.Fatalf("resolved screen = %#v, ok=%v", resolved, ok)
	}
	want := application.Rect{X: 460, Y: 170, Width: 1000, Height: 700}
	if bounds != want {
		t.Fatalf("bounds = %#v, want %#v", bounds, want)
	}
}

func TestResolveClampsCorruptOrStaleBoundsToWorkArea(t *testing.T) {
	primary := testScreen("primary", "primary", true, application.Rect{X: 0, Y: 30, Width: 800, Height: 570})
	saved := Placement{ScreenID: primary.ID, X: 9000, Y: -9000, Width: 4000, Height: 4000}

	bounds, _, ok := Resolve(saved, []*application.Screen{primary}, 1080, 720, 560, 560)
	if !ok {
		t.Fatal("placement was not resolved")
	}
	want := primary.WorkArea
	if bounds != want {
		t.Fatalf("bounds = %#v, want %#v", bounds, want)
	}
}

func TestCenterOverClampsChildToOwnerWorkArea(t *testing.T) {
	workArea := application.Rect{X: 1920, Y: 0, Width: 1280, Height: 720}
	owner := application.Rect{X: 3000, Y: 650, Width: 400, Height: 300}

	got := CenterOver(owner, 620, 440, workArea)
	want := application.Rect{X: 2580, Y: 280, Width: 620, Height: 440}
	if got != want {
		t.Fatalf("bounds = %#v, want %#v", got, want)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), stateFileName)}
	want := Placement{ScreenID: "id", ScreenName: "name", X: 12, Y: 34, Width: 1080, Height: 720}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !found || got != want {
		t.Fatalf("placement = %#v, found=%v, want %#v", got, found, want)
	}
}

func testScreen(id, name string, primary bool, workArea application.Rect) *application.Screen {
	return &application.Screen{ID: id, Name: name, IsPrimary: primary, WorkArea: workArea, Bounds: workArea, Size: application.Size{Width: workArea.Width, Height: workArea.Height}}
}
