package managedruntime

// platformRecipe is acquisition/host policy, separate from provider/model
// behavior. Only qualified Windows x64 artifacts are registered. Future hosts
// must supply their own executable layout, dependencies and accelerator policy.
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

// Compatibility projection for existing provider consumers; pins live only in
// the table above. Admission still resolves the actual host against the table.
func windowsCPURecipe(provider ProviderID) platformRecipe {
	r, _ := recipeFor(provider, "windows", "amd64", "cpu")
	r.release = r.archives[0]
	cuda, _ := recipeFor(provider, "windows", "amd64", "cuda")
	r.cuda = cuda.runtimeBundle
	return r
}
