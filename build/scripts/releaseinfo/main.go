// Command releaseinfo synchronizes the generated Windows version assets with
// build/config.yml. It deliberately owns only version fields; Wails remains
// responsible for generating the surrounding platform/package templates.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tnware/freehand-stt/internal/releaseinfo"
)

type versionResource struct {
	Fixed struct {
		FileVersion string `json:"file_version"`
	} `json:"fixed"`
	Info map[string]map[string]string `json:"info"`
}

type asset struct {
	path      string
	transform func([]byte) ([]byte, error)
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	mode := "check"
	if flag.NArg() == 1 {
		mode = flag.Arg(0)
	}
	if flag.NArg() > 1 || (mode != "check" && mode != "sync") {
		fatal(errors.New("usage: releaseinfo [-root path] [check|sync]"))
	}
	configPath := filepath.Join(*root, "build", "config.yml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		fatal(fmt.Errorf("read release source: %w", err))
	}
	info, err := releaseinfo.Parse(config)
	if err != nil {
		fatal(err)
	}

	assets := []asset{
		{
			path: filepath.Join(*root, "build", "windows", "info.json"),
			transform: func(data []byte) ([]byte, error) {
				return versionedJSON(data, info)
			},
		},
		{
			path: filepath.Join(*root, "build", "windows", "wails.exe.manifest"),
			transform: func(data []byte) ([]byte, error) {
				pattern := regexp.MustCompile(`(<assemblyIdentity\b[^>]*\bname="` + regexp.QuoteMeta(info.ProductIdentifier) + `"[^>]*\bversion=")[^"]+(")`)
				return replaceOne(data, pattern, `${1}`+info.WindowsVersion+`${2}`, "application assembly version")
			},
		},
		{
			path: filepath.Join(*root, "build", "windows", "nsis", "project.nsi"),
			transform: func(data []byte) ([]byte, error) {
				pattern := regexp.MustCompile(`(?m)(!define INFO_BINARYVERSION ")[^"]+(")`)
				return replaceOne(data, pattern, `${1}`+info.WindowsVersion+`${2}`, "NSIS binary version")
			},
		},
		{
			path: filepath.Join(*root, "build", "windows", "nsis", "wails_tools.nsh"),
			transform: func(data []byte) ([]byte, error) {
				pattern := regexp.MustCompile(`(?m)(!define INFO_PRODUCTVERSION ")[^"]+(")`)
				return replaceOne(data, pattern, `${1}`+info.Version+`${2}`, "NSIS display version")
			},
		},
	}

	var stale []string
	for _, candidate := range assets {
		current, err := os.ReadFile(candidate.path)
		if err != nil {
			fatal(fmt.Errorf("read %s: %w", candidate.path, err))
		}
		expected, err := candidate.transform(current)
		if err != nil {
			fatal(fmt.Errorf("synchronize %s: %w", candidate.path, err))
		}
		if bytes.Equal(current, expected) {
			continue
		}
		if mode == "check" {
			stale = append(stale, candidate.path)
			continue
		}
		if err := os.WriteFile(candidate.path, expected, 0o644); err != nil {
			fatal(fmt.Errorf("write %s: %w", candidate.path, err))
		}
		fmt.Printf("updated %s\n", candidate.path)
	}
	if len(stale) > 0 {
		fatal(fmt.Errorf("release version assets are stale: %s; run wails3 task common:update:build-assets", strings.Join(stale, ", ")))
	}
	fmt.Printf("release identity %s (Windows %s) is synchronized\n", info.Version, info.WindowsVersion)
}

func versionedJSON(data []byte, info releaseinfo.Info) ([]byte, error) {
	var resource versionResource
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, errors.New("Windows version resource is not valid JSON")
	}
	if len(resource.Info) == 0 {
		return nil, errors.New("Windows version resource has no language entries")
	}
	resource.Fixed.FileVersion = info.WindowsVersion
	for _, values := range resource.Info {
		values["ProductVersion"] = info.Version
		values["FileVersion"] = info.Version
	}
	result, err := json.MarshalIndent(resource, "", "\t")
	if err != nil {
		return nil, errors.New("Windows version resource could not be serialized")
	}
	result = append(result, '\n')
	if bytes.Contains(data, []byte("\r\n")) {
		result = bytes.ReplaceAll(result, []byte("\n"), []byte("\r\n"))
	}
	return result, nil
}

func replaceOne(data []byte, pattern *regexp.Regexp, replacement, label string) ([]byte, error) {
	if matches := pattern.FindAllIndex(data, -1); len(matches) != 1 {
		return nil, fmt.Errorf("expected exactly one %s, found %d", label, len(matches))
	}
	return pattern.ReplaceAll(data, []byte(replacement)), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "releaseinfo:", err)
	os.Exit(1)
}
