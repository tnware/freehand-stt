//go:build windows || darwin

package managedruntime

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (m *Manager) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return m.startup(ctx)
}
