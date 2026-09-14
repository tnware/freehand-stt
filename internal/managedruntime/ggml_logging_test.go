package managedruntime

import (
	"slices"
	"strings"
	"testing"
)

func TestLlamaLoggingArguments(t *testing.T) {
	if !llamaProvider.descriptor().Supported {
		t.Skip("Windows x64 recipe")
	}
	for _, backend := range []string{"cpu", "cuda"} {
		t.Run(backend, func(t *testing.T) {
			args, err := llamaProvider.backendArguments("s1-mini", llamaProvider.specs["s1-mini"], t.TempDir(), 1234, backend)
			if err != nil {
				t.Fatal(err)
			}
			// b10809: level 3 is normal info/warnings/errors, not trace/debug.
			// An exact logging-option allowlist also forbids disabling output,
			// file/prompt logging, and alternative verbosity aliases/overrides.
			var logging []string
			for i := 0; i < len(args); i++ {
				arg := args[i]
				if !strings.HasPrefix(arg, "--log-") && arg != "-v" && arg != "--verbose" && arg != "-lv" && arg != "--verbosity" {
					continue
				}
				logging = append(logging, arg)
				if (arg == "--log-verbosity" || arg == "--log-colors") && i+1 < len(args) {
					i++
					logging = append(logging, args[i])
				}
			}
			if !slices.Equal(logging, []string{"--log-verbosity", "3", "--log-colors", "on"}) {
				t.Fatalf("expected normal private console logging only, got %v", logging)
			}
		})
	}
}

func TestLlamaLoggingEnvironmentRejectsOverrides(t *testing.T) {
	inherited := []string{
		"SystemRoot=C:\\Windows", "ProgramFiles=C:\\Program Files", "TEMP=C:\\Temp", "TMP=C:\\Temp", "WINDIR=C:\\Windows",
		"LLAMA_ARG_LOG_FILE=private.log", "LLAMA_ARG_LOG_VERBOSITY=5", "LLAMA_ARG_LOG_COLORS=on",
		"LLAMA_ARG_LOG_PREFIX=1", "LLAMA_ARG_LOG_TIMESTAMPS=1",
		// b10809 discovers config.ini through these Windows locations before
		// parsing argv. Neither may re-enable file logging or other overrides.
		"APPDATA=C:\\ForeignUserConfig", "PROGRAMDATA=C:\\ForeignSystemConfig",
		"HF_TOKEN=secret", "HTTPS_PROXY=foreign", "PATH=foreign", "GGML_BACKEND_PATH=foreign",
	}
	if got := ggmlEnvironment(inherited); !slices.Equal(got, inherited[:5]) {
		t.Fatal("logging/config/credential overrides escaped the OS environment allowlist")
	}
}
