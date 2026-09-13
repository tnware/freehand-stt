package input

import (
	"context"
	"errors"
	"testing"
)

type permissionFake struct {
	calls []string
	ctx   context.Context
}

func (p *permissionFake) Current() PermissionStatus {
	return PermissionStatus{Required: true, Microphone: "authorized", Accessibility: true, Keyboard: true}
}
func (p *permissionFake) Request(ctx context.Context, kind string) error {
	p.calls = append(p.calls, kind)
	p.ctx = ctx
	return ctx.Err()
}
func (p *permissionFake) OpenSettings(kind string) error {
	p.calls = append(p.calls, "open:"+kind)
	return nil
}

func TestNativePermissionsAreReadOnlyAndRequestsValidated(t *testing.T) {
	p := &permissionFake{}
	s := &Service{}
	ConfigurePermissions(s, p)
	if got := s.NativePermissions(); !got.Required || got.Microphone != "authorized" {
		t.Fatalf("status %#v", got)
	}
	if len(p.calls) != 0 {
		t.Fatal("status requested a permission")
	}
	for _, kind := range []string{"", "camera", "microphone?redirect=evil"} {
		if _, err := s.RequestPermission(kind); err == nil {
			t.Fatalf("accepted %q", kind)
		}
		if err := s.OpenPermissionSettings(kind); err == nil {
			t.Fatalf("opened %q", kind)
		}
	}
	if len(p.calls) != 0 {
		t.Fatal("invalid requests reached native adapter")
	}
	if _, err := s.RequestPermission("microphone"); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.ctx.Deadline(); !ok {
		t.Fatal("request lacks deadline")
	}
	if err := s.OpenPermissionSettings("accessibility"); err != nil {
		t.Fatal(err)
	}
}

func TestPermissionRequestsCancelWithApplicationAndRejectAfterClose(t *testing.T) {
	p := &permissionFake{}
	root, cancel := context.WithCancel(context.Background())
	s := &Service{rootContext: root}
	ConfigurePermissions(s, p)
	cancel()
	if _, err := s.RequestPermission("microphone"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	s.closed.Store(true)
	before := len(p.calls)
	if _, err := s.RequestPermission("accessibility"); err == nil {
		t.Fatal("request after shutdown")
	}
	if err := s.OpenPermissionSettings("keyboard"); err == nil {
		t.Fatal("open after shutdown")
	}
	if len(p.calls) != before {
		t.Fatal("closed service called native adapter")
	}
}
