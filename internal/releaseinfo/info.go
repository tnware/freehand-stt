// Package releaseinfo parses and validates the repository's authoritative
// release identity. build/config.yml is embedded into the executable and is
// also the source consumed by Wails when it generates packaging assets.
package releaseinfo

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	maxProductNameBytes       = 80
	maxProductIdentifierBytes = 200
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// Info is the validated subset of build/config.yml needed by the running app
// and by version-asset synchronization.
type Info struct {
	ProductName       string
	ProductIdentifier string
	Version           string
	WindowsVersion    string
}

type sourceConfig struct {
	Info struct {
		ProductName       string `yaml:"productName"`
		ProductIdentifier string `yaml:"productIdentifier"`
		Version           string `yaml:"version"`
	} `yaml:"info"`
}

// Parse returns a strict release identity. A malformed embedded source is a
// build/configuration failure rather than something the app should silently
// label with an invented version.
func Parse(data []byte) (Info, error) {
	var source sourceConfig
	if err := yaml.Unmarshal(data, &source); err != nil {
		return Info{}, errors.New("release configuration is not valid YAML")
	}
	name := strings.TrimSpace(source.Info.ProductName)
	identifier := strings.TrimSpace(source.Info.ProductIdentifier)
	version := strings.TrimSpace(source.Info.Version)
	if name == "" || len(name) > maxProductNameBytes || hasControl(name) {
		return Info{}, errors.New("release product name is invalid")
	}
	if identifier == "" || len(identifier) > maxProductIdentifierBytes || hasControl(identifier) {
		return Info{}, errors.New("release product identifier is invalid")
	}
	windowsVersion, err := WindowsVersion(version)
	if err != nil {
		return Info{}, err
	}
	return Info{
		ProductName:       name,
		ProductIdentifier: identifier,
		Version:           version,
		WindowsVersion:    windowsVersion,
	}, nil
}

// WindowsVersion derives the four numeric uint16 components required by PE,
// assembly manifests, and NSIS. Stable SemVer uses revision zero; a prerelease
// must end in a positive numeric sequence such as alpha.1 or rc.3.
func WindowsVersion(version string) (string, error) {
	matches := semverPattern.FindStringSubmatch(strings.TrimSpace(version))
	if matches == nil {
		return "", errors.New("release version must be valid semantic versioning")
	}
	parts := make([]uint64, 4)
	for index := 0; index < 3; index++ {
		value, err := strconv.ParseUint(matches[index+1], 10, 16)
		if err != nil {
			return "", fmt.Errorf("release version component %d exceeds the Windows limit", index+1)
		}
		parts[index] = value
	}
	if prerelease := matches[4]; prerelease != "" {
		identifiers := strings.Split(prerelease, ".")
		revision, err := strconv.ParseUint(identifiers[len(identifiers)-1], 10, 16)
		if err != nil || revision == 0 {
			return "", errors.New("prerelease version must end in a positive numeric revision")
		}
		parts[3] = revision
	}
	return fmt.Sprintf("%d.%d.%d.%d", parts[0], parts[1], parts[2], parts[3]), nil
}

func hasControl(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f })
}
