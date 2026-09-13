//go:build windows || darwin

package managedruntime

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return s.startup(ctx)
}
