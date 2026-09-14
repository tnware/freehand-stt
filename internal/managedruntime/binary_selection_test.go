package managedruntime

import (
	"slices"
	"testing"
)

func TestBinarySelectionCUDAWarmup(t *testing.T) {
	if !llamaProvider.descriptor().Supported {
		t.Skip("Windows x64 recipe")
	}
	for _, backend := range []string{"cpu", "cuda"} {
		args, err := llamaProvider.backendArguments("s1-mini", llamaProvider.specs["s1-mini"], t.TempDir(), 1234, backend)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(args, "--no-warmup") != (backend == "cpu") {
			t.Fatalf("%s args=%v", backend, args)
		}
	}
}

func TestBinarySelectionHostAndMetadata(t *testing.T) {
	for _, provider := range []ProviderID{LlamaCPP, WhisperCPP} {
		for _, tc := range []struct {
			name, os, arch, vendor, metadata, want string
			ordinal                                int
			known, supported                       bool
		}{
			{"qualified", "windows", "amd64", "nvidia", "551.78, 5.0", "cuda", 0, true, true},
			{"old driver", "windows", "amd64", "nvidia", "551.77, 8.6", "cpu", 0, true, true},
			{"unknown", "windows", "amd64", "", "", "cpu", 0, false, true},
			{"malformed", "windows", "amd64", "nvidia", "NaN, 8.6", "cpu", 0, true, true},
			{"multiple GPUs", "windows", "amd64", "nvidia", "551.78, 8.6\n551.78, 8.6", "cpu", 0, true, true},
			{"old GPU", "windows", "amd64", "nvidia", "551.78, 3.5", "cpu", 0, true, true},
			{"wrong vendor", "windows", "amd64", "amd", "610.62, 12.0", "cpu", 0, true, true},
			{"wrong ordinal", "windows", "amd64", "nvidia", "610.62, 12.0", "cpu", 1, true, true},
			{"mac extension", "darwin", "arm64", "apple", "", "", 0, true, false},
			{"windows arm", "windows", "arm64", "nvidia", "610.62, 12.0", "", 0, true, false},
		} {
			t.Run(string(provider)+tc.name, func(t *testing.T) {
				got, err := selectBinaryOptions(provider, binaryHost{os: tc.os, arch: tc.arch, gpu: gpuMetadata{vendor: tc.vendor, ordinal: tc.ordinal, output: tc.metadata, known: tc.known}})
				if err != nil || got.RecommendedBackend != tc.want || got.Supported != tc.supported || got.Reason == "" || len(got.Options) != 2 {
					t.Fatalf("options=%+v err=%v", got, err)
				}
				if got.Options[1].Available != (tc.want == "cuda") {
					t.Fatalf("CUDA admission disagrees: %+v", got)
				}
			})
		}
	}
	if _, err := selectBinaryOptions(ProviderID("../llama"), binaryHost{}); err == nil {
		t.Fatal("unknown provider accepted")
	}
}
