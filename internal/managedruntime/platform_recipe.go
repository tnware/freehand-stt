package managedruntime

import "runtime"

// platformRecipe is acquisition/host policy, separate from provider/model
// behavior. Each host supplies its executable layout and accelerator policy.
type platformRecipe struct {
	// Derived compatibility access for existing adapter fixtures/consumers.
	// Never initialize pins here: recipeFor remains the sole pin registry.
	release asset
	cuda    runtimeBundle
	version string
	runtimeBundle
	minDriver, minCapability float64
}
type platformRecipeKey struct {
	provider          ProviderID
	os, arch, backend string
}

var platformRecipes = map[platformRecipeKey]platformRecipe{
	{NeMoSpeechCPP, "windows", "amd64", "cpu"}:    {version: Version, runtimeBundle: runtimeBundle{executable: "bin/nemo-speech.exe", archives: []asset{assets["cpu"]}}},
	{NeMoSpeechCPP, "windows", "amd64", "cuda"}:   {version: Version, runtimeBundle: runtimeBundle{executable: "bin/nemo-speech.exe", archives: []asset{assets["cuda"]}}},
	{NeMoSpeechCPP, "windows", "amd64", "vulkan"}: {version: Version, runtimeBundle: runtimeBundle{executable: "bin/nemo-speech.exe", archives: []asset{assets["vulkan"]}}},
	{NeMoSpeechCPP, "darwin", "arm64", "cpu"}: {version: Version, runtimeBundle: runtimeBundle{executable: "nemo-speech/bin/nemo-speech", archives: []asset{
		{"cpu", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-macos-aarch64-cpu.tar.gz", "971661d38d4bf97a63c528d13041a964316d25068d8df045e5b4839848092f25", 3299661},
	}}},
	{NeMoSpeechCPP, "darwin", "arm64", "metal"}: {version: Version, runtimeBundle: runtimeBundle{executable: "nemo-speech/bin/nemo-speech", archives: []asset{
		{"metal", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-macos-aarch64-metal.tar.gz", "f1dff4f9dd9c96214f8cb78b982812459132df8a4ad1a42409fd94de4a366244", 3465028},
	}}},
	{NeMoSpeechCPP, "darwin", "amd64", "cpu"}: {version: Version, runtimeBundle: runtimeBundle{executable: "nemo-speech/bin/nemo-speech", archives: []asset{
		{"cpu", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-macos-x86_64-cpu.tar.gz", "042a4612e07460fab6a39b5d862aa1e39d0ac3eaedfdb979f3f5fc12de510c20", 3618245},
	}}},
	{LlamaCPP, "darwin", "arm64", "cpu"}: {version: "b10809", runtimeBundle: runtimeBundle{executable: "llama-b10809/llama-server", archives: []asset{
		{"cpu", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-macos-arm64.tar.gz", "7d692df9e1e386e62f1c12b843903218041e6cd74c9415aa39a7ed3176f9eaa2", 11123196},
	}}},
	{LlamaCPP, "darwin", "arm64", "metal"}: {version: "b10809", runtimeBundle: runtimeBundle{executable: "llama-b10809/llama-server", archives: []asset{
		{"metal", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-macos-arm64.tar.gz", "7d692df9e1e386e62f1c12b843903218041e6cd74c9415aa39a7ed3176f9eaa2", 11123196},
	}}},
	{LlamaCPP, "darwin", "amd64", "cpu"}: {version: "b10809", runtimeBundle: runtimeBundle{executable: "llama-b10809/llama-server", archives: []asset{
		{"cpu", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-macos-x64.tar.gz", "13b34aa8a5d87341a21065a83f54a8167e1aaa6fe0d66065de01632a1ed64be6", 11175330},
	}}},
	{LlamaCPP, "windows", "amd64", "cpu"}: {version: "b10809", runtimeBundle: runtimeBundle{executable: "llama-server.exe", archives: []asset{
		{"cpu", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-win-cpu-x64.zip", "9df3158ed228a641a4b127942d7f459f24c9e13f04682659d05c00c80099b6b5", 18407457},
	}}},
	{LlamaCPP, "windows", "amd64", "cuda"}: {version: "b10809", minDriver: 551.78, minCapability: 5.0, runtimeBundle: runtimeBundle{executable: "llama-server.exe", required: []string{"ggml-cuda.dll", "cudart64_12.dll", "cublas64_12.dll", "cublasLt64_12.dll"}, archives: []asset{
		{"cuda", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-win-cuda-12.4-x64.zip", "c77bfcd9ed8d91e8721a2d6a290b907fddd4fa5412a47b21c6fa1709116b85f9", 253938543},
		{"cuda", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/cudart-llama-bin-win-cuda-12.4-x64.zip", "8c79a9b226de4b3cacfd1f83d24f962d0773be79f1e7b75c6af4ded7e32ae1d6", 391443627},
	}}},
	{WhisperCPP, "windows", "amd64", "cpu"}: {version: "v1.8.3", runtimeBundle: runtimeBundle{executable: "Release/whisper-server.exe", archives: []asset{
		{"cpu", "https://github.com/ggml-org/whisper.cpp/releases/download/v1.8.3/whisper-bin-x64.zip", "d824b1e37599f882b396e73f1ee0bfd5d0529f700314c48311dcbd00b803321d", 3968674},
	}}},
	{WhisperCPP, "windows", "amd64", "cuda"}: {version: "v1.8.3", minDriver: 551.78, minCapability: 5.0, runtimeBundle: runtimeBundle{executable: "Release/whisper-server.exe", required: []string{"Release/ggml-cuda.dll", "Release/cudart64_12.dll", "Release/cublas64_12.dll", "Release/cublasLt64_12.dll"}, archives: []asset{
		{"cuda", "https://github.com/ggml-org/whisper.cpp/releases/download/v1.8.3/whisper-cublas-12.4.0-bin-x64.zip", "c12a563333d3c3707be70754dc0e87c1cb58aa6333a87055bbcf9b524488dfb0", 459854042},
	}}},
}

func recipeFor(provider ProviderID, os, arch, backend string) (platformRecipe, bool) {
	r, ok := platformRecipes[platformRecipeKey{provider, os, arch, backend}]
	return r, ok
}

func hostBackends(provider ProviderID) []string {
	backends := []string{}
	for _, backend := range []string{"cpu", "cuda", "metal", "vulkan"} {
		if _, ok := recipeFor(provider, runtime.GOOS, runtime.GOARCH, backend); ok {
			backends = append(backends, backend)
		}
	}
	return backends
}

// Compatibility projection for existing provider consumers; pins live only in
// the table above. Admission still resolves the actual host against the table.
func hostCPURecipe(provider ProviderID) platformRecipe {
	r, ok := recipeFor(provider, runtime.GOOS, runtime.GOARCH, "cpu")
	if !ok {
		// Retain metadata for unavailable providers; admission still resolves
		// the actual host and cannot install this fallback recipe.
		r, _ = recipeFor(provider, "windows", "amd64", "cpu")
	}
	r.release = r.archives[0]
	cuda, _ := recipeFor(provider, "windows", "amd64", "cuda")
	r.cuda = cuda.runtimeBundle
	return r
}
