package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"testing"
)

type iconProbe struct{ kinds []string }

func (p *iconProbe) SetIcon([]byte) *application.SystemTray {
	p.kinds = append(p.kinds, "light")
	return nil
}
func (p *iconProbe) SetDarkModeIcon([]byte) *application.SystemTray {
	p.kinds = append(p.kinds, "dark")
	return nil
}
func (p *iconProbe) SetTemplateIcon([]byte) *application.SystemTray {
	p.kinds = append(p.kinds, "template")
	return nil
}
func TestMacTrayUsesTemplateNotICO(t *testing.T) {
	p := &iconProbe{}
	installIcons(p, Icons{Light: []byte{1}, Dark: []byte{2}}, "darwin")
	if len(p.kinds) != 1 || p.kinds[0] != "template" {
		t.Fatalf("Mac icons=%v", p.kinds)
	}
}
func TestWindowsTrayKeepsLightAndDarkIcons(t *testing.T) {
	p := &iconProbe{}
	installIcons(p, Icons{Light: []byte{1}, Dark: []byte{2}}, "windows")
	if len(p.kinds) != 2 || p.kinds[0] != "light" || p.kinds[1] != "dark" {
		t.Fatalf("Windows icons=%v", p.kinds)
	}
}
