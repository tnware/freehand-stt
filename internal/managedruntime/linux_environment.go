package managedruntime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
)

// b10809 applies system and user config before argv. Linux falls back to the
// passwd home if HOME is absent. Never let ambient configuration enable file
// logging, remote resources, or alternate model/backend paths in managed mode.
func managedGGMLEnvironment(root string, provider ProviderID, inherited []string) ([]string, error) {
	env := ggmlEnvironment(inherited)
	if runtime.GOOS != "linux" {
		return env, nil
	}
	if provider == LlamaCPP {
		if err := requireNoLlamaSystemConfig("/etc/llama.cpp/config.ini"); err != nil {
			return nil, err
		}
	}
	home := filepath.Join(root, "private-home")
	if err := artifact.CheckRoot(home); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, err
	}
	config := filepath.Join(home, ".config")
	if err := requireNoLlamaSystemConfig(filepath.Join(config, "llama.cpp", "config.ini")); err != nil {
		return nil, err
	}
	return append(env, "HOME="+home, "XDG_CONFIG_HOME="+config, "XDG_CACHE_HOME="+filepath.Join(home, ".cache")), nil
}

func requireNoLlamaSystemConfig(filename string) error {
	_, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	// Existing files, dangling symlinks, and permission/IO failures all block.
	return errors.New("Managed llama.cpp cannot start while an external configuration file exists or cannot be checked. Remove the configuration or use a manually managed server.")
}

// The pinned NeMo downloader invokes curl without --disable. On Linux curl
// can discover the passwd home even when HOME was omitted; provide an empty,
// private location and reject unexpected config before launching its CLI.
func prepareNeMoLinuxEnvironment(root string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	home := filepath.Join(root, "private-home")
	if err := artifact.CheckRoot(home); err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return err
	}
	for _, filename := range []string{filepath.Join(home, ".curlrc"), filepath.Join(home, ".config", "curlrc")} {
		if _, err := os.Lstat(filename); !errors.Is(err, os.ErrNotExist) {
			return errors.New("Managed speech cannot use an unexpected curl configuration. Remove the private runtime configuration and try again.")
		}
	}
	return nil
}
