package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
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
	}
	if err := yaml.Unmarshal(data, &source); err != nil {
		return macInfo{}, errors.New("invalid macOS release configuration")
	}
	build, err := macBuildVersion(identity)
	if err != nil {
		return macInfo{}, err
	}
	return macInfo{Info: identity, BuildNumber: build, Description: source.Info.Description, Copyright: source.Info.Copyright}, nil
}

// macBuildVersion derives Apple's three numeric components from the same
// release version Release Please updates. Each patch gets four 16-bit slots:
// alpha, beta, rc, then stable. This preserves release ordering without a second
// manually incremented build number; the marketing version remains unchanged.
// https://developer.apple.com/documentation/bundleresources/information-property-list/cfbundleversion
func macBuildVersion(info releaseinfo.Info) (string, error) {
	parts := strings.Split(info.WindowsVersion, ".") // validated uint16 components
	patch, _ := strconv.ParseUint(parts[2], 10, 16)
	revision, _ := strconv.ParseUint(parts[3], 10, 16)
	stage := uint64(3)
	version := strings.SplitN(info.Version, "+", 2)[0]
	if _, prerelease, ok := strings.Cut(version, "-"); ok {
		identifiers := strings.Split(prerelease, ".")
		if len(identifiers) != 2 {
			return "", errors.New("macOS prerelease must use alpha.N, beta.N or rc.N")
		}
		switch identifiers[0] {
		case "alpha":
			stage = 0
		case "beta":
			stage = 1
		case "rc":
			stage = 2
		default:
			return "", errors.New("macOS prerelease must use alpha.N, beta.N or rc.N")
		}
	}
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch*(1<<18)+stage*(1<<16)+revision), nil
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
