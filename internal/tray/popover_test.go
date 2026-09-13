package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"reflect"
	"testing"
)

type popoverTrayFixture struct {
	calls    []string
	click    func()
	right    func()
	attached application.Window
	offset   int
}

func (f *popoverTrayFixture) AttachWindow(w application.Window) *application.SystemTray {
	f.attached = w
	f.calls = append(f.calls, "attach")
	return nil
}
func (f *popoverTrayFixture) OnClick(fn func()) *application.SystemTray {
	f.click = fn
	f.calls = append(f.calls, "left")
	return nil
}
func (f *popoverTrayFixture) OnRightClick(fn func()) *application.SystemTray {
	f.right = fn
	f.calls = append(f.calls, "right")
	return nil
}
func (f *popoverTrayFixture) ToggleWindow() { f.calls = append(f.calls, "toggle") }
func (f *popoverTrayFixture) HideWindow()   { f.calls = append(f.calls, "hide") }
func (f *popoverTrayFixture) ShowMenu()     { f.calls = append(f.calls, "menu") }

func (f *popoverTrayFixture) WindowOffset(offset int) *application.SystemTray {
	f.offset = offset
	return nil
}

func TestAttachPopoverWindowsEdgeSpacing(t *testing.T) {
	for _, osName := range []string{"windows", "darwin", "linux"} {
		t.Run(osName, func(t *testing.T) {
			f := &popoverTrayFixture{}
			attachPopover(osName, f, &application.WebviewWindow{})
			want := 0
			if osName == "windows" {
				want = 8
			}
			if f.offset != want {
				t.Fatalf("window offset = %d, want %d", f.offset, want)
			}
		})
	}
}

func TestAttachPopoverWindowsDismissesBeforeNativeMenu(t *testing.T) {
	f := &popoverTrayFixture{}
	w := &application.WebviewWindow{}
	attachPopover("windows", f, w)
	if f.attached != w || f.click == nil || f.right == nil {
		t.Fatal("Windows must attach the panel and retain both click actions")
	}
	f.click()
	f.right()
	if !reflect.DeepEqual(f.calls, []string{"attach", "left", "right", "toggle", "hide", "menu"}) {
		t.Fatalf("panel/menu ordering = %v", f.calls)
	}
}

func TestAttachPopoverMacAndUnsupportedPlatform(t *testing.T) {
	for _, osName := range []string{"darwin", "linux"} {
		t.Run(osName, func(t *testing.T) {
			f := &popoverTrayFixture{right: func() {}}
			w := &application.WebviewWindow{}
			attachPopover(osName, f, w)
			if osName != "darwin" {
				if len(f.calls) != 0 {
					t.Fatal(f.calls)
				}
				return
			}
			if f.attached != w || f.click == nil || f.right != nil {
				t.Fatal("attachment must override left click and restore native right menu")
			}
			f.click()
			if !reflect.DeepEqual(f.calls, []string{"attach", "left", "right", "toggle"}) {
				t.Fatal(f.calls)
			}
		})
	}
}
