// Package buildinfo exposes immutable, renderer-safe application build
// metadata. It performs no lifecycle work and never includes host, path,
// credential, endpoint, or user data.
package buildinfo

import (
	"runtime"
	"runtime/debug"
	"strings"
)

const wailsModule = "github.com/wailsapp/wails/v3"

// Info is the bounded build identity shown in About.
type Info struct {
	ProductName    string `json:"productName"`
	Version        string `json:"version"`
	WindowsVersion string `json:"windowsVersion"`
	Development    bool   `json:"development"`
	GoVersion      string `json:"goVersion"`
	WailsVersion   string `json:"wailsVersion"`
}

type Service struct {
	info Info
}

func NewService(productName, version, windowsVersion string, development bool) *Service {
	return &Service{info: Info{
		ProductName:    productName,
		Version:        version,
		WindowsVersion: windowsVersion,
		Development:    development,
		GoVersion:      strings.TrimPrefix(runtime.Version(), "go"),
		WailsVersion:   dependencyVersion(wailsModule),
	}}
}

// Current returns the immutable build identity for this executable.
func (s *Service) Current() Info { return s.info }

func dependencyVersion(module string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dependency := range info.Deps {
		if dependency.Path == module {
			return strings.TrimPrefix(dependency.Version, "v")
		}
	}
	return "unknown"
}
