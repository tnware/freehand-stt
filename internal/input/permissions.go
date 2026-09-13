package input

import (
	"context"
	"errors"
	"time"
)

// PermissionStatus reports only bounded OS authorization state, never prompts.
type PermissionStatus struct {
	Required      bool   `json:"required"`
	Microphone    string `json:"microphone"`
	Accessibility bool   `json:"accessibility"`
	Keyboard      bool   `json:"keyboard"`
}

// PermissionAccess is the native capability injected before service startup.
type PermissionAccess interface {
	Current() PermissionStatus
	Request(context.Context, string) error
	OpenSettings(string) error
}

func ConfigurePermissions(s *Service, access PermissionAccess) { s.permissions = access }

func (s *Service) NativePermissions() PermissionStatus {
	if s.permissions == nil {
		return PermissionStatus{Microphone: "not-required"}
	}
	return s.permissions.Current()
}

func validPermission(kind string) bool {
	return kind == "microphone" || kind == "accessibility" || kind == "keyboard"
}

// RequestPermission runs only from an explicit user action, not inventory or startup.
func (s *Service) RequestPermission(kind string) (PermissionStatus, error) {
	if !validPermission(kind) {
		return PermissionStatus{}, errors.New("unknown native permission")
	}
	if s.closed.Load() {
		return PermissionStatus{}, errors.New("application is shutting down")
	}
	if s.permissions == nil {
		return PermissionStatus{}, errors.New("native permission requests are unavailable")
	}
	if !s.permissionMu.TryLock() {
		return PermissionStatus{}, errors.New("a permission request is already active")
	}
	defer s.permissionMu.Unlock()
	ctx, cancel := s.operationContext(90 * time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return PermissionStatus{}, err
	}
	if err := s.permissions.Request(ctx, kind); err != nil {
		return s.NativePermissions(), err
	}
	return s.NativePermissions(), nil
}

func (s *Service) OpenPermissionSettings(kind string) error {
	if !validPermission(kind) {
		return errors.New("unknown native permission")
	}
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	if s.permissions == nil {
		return errors.New("native permission settings are unavailable")
	}
	return s.permissions.OpenSettings(kind)
}
