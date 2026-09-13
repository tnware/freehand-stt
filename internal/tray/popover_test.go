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

func TestAttachPopoverMacOnly(t *testing.T) {
	for _, osName := range []string{"darwin", "windows", "linux"} {
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
