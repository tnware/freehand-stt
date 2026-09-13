package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tnware/freehand-stt/internal/releaseinfo"
	"gopkg.in/yaml.v3"
)

type macInfo struct {
	releaseinfo.Info
	BuildNumber, Description, Copyright string
}

func parseMacInfo(data []byte) (macInfo, error) {
	identity, err := releaseinfo.Parse(data)
	if err != nil {
		return macInfo{}, err
	}
	var source struct {
		Info struct {
			Description string `yaml:"description"`
			Copyright   string `yaml:"copyright"`
		} `yaml:"info"`
		MacOS struct {
			BuildNumber string `yaml:"buildNumber"`
		} `yaml:"macos"`
	}
	if err := yaml.Unmarshal(data, &source); err != nil {
		return macInfo{}, errors.New("invalid macOS release configuration")
	}
	if !regexp.MustCompile(`^[1-9][0-9]{0,3}$`).MatchString(source.MacOS.BuildNumber) {
		return macInfo{}, errors.New("macOS buildNumber must be a positive integer from 1 to 9999")
	}
	return macInfo{Info: identity, BuildNumber: source.MacOS.BuildNumber, Description: source.Info.Description, Copyright: source.Info.Copyright}, nil
}

// macPlist owns both generated plists. A distinct development identity avoids
// sharing microphone/accessibility grants with the ad-hoc production bundle.
func macPlist(info macInfo, dev bool) ([]byte, error) {
	id := info.ProductIdentifier
	if dev {
		id += ".dev"
	}
	var out bytes.Buffer
	out.WriteString(xml.Header)
	out.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n<plist version=\"1.0\">\n\t<dict>\n")
	values := [][2]string{
		{"CFBundleExecutable", "freehand"},
		{"CFBundleGetInfoString", info.Description},
		{"CFBundleIconFile", "icons"},
		{"CFBundleIconName", "appicon"},
		{"CFBundleIdentifier", id},
		{"CFBundleName", info.ProductName},
		{"CFBundleDisplayName", info.ProductName},
		{"CFBundlePackageType", "APPL"},
		{"CFBundleShortVersionString", strings.SplitN(strings.SplitN(info.Version, "-", 2)[0], "+", 2)[0]},
		{"CFBundleVersion", info.BuildNumber},
		{"FreehandReleaseVersion", info.Version},
		{"LSMinimumSystemVersion", "13.0"},
		{"NSMicrophoneUsageDescription", "Freehand records your microphone only when you start dictation and sends audio to your chosen speech endpoint."},
		{"NSHumanReadableCopyright", info.Copyright},
	}
	for _, pair := range values {
		fmt.Fprintf(&out, "\t\t<key>%s</key>\n\t\t<string>", pair[0])
		if err := xml.EscapeText(&out, []byte(pair[1])); err != nil {
			return nil, errors.New("macOS release metadata cannot be encoded")
		}
		out.WriteString("</string>\n")
	}
	out.WriteString("\t\t<key>NSHighResolutionCapable</key>\n\t\t<true/>\n")
	if dev {
		out.WriteString("\t\t<key>NSAppTransportSecurity</key>\n\t\t<dict>\n\t\t\t<key>NSAllowsLocalNetworking</key>\n\t\t\t<true/>\n\t\t</dict>\n")
	}
	out.WriteString("\t</dict>\n</plist>\n")
	return out.Bytes(), nil
}
