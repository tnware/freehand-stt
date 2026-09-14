// Managed installation choices, not API compatibility flags or an upstream
// model inventory. Keep aligned with internal/managedruntime/provider_*.go
// and platform_recipe.go. The setup guide owns detailed requirements.
export const managedRuntimes = [
  {
    backend: "nemo-speech-v1",
    name: "NeMo-Speech.cpp",
    platforms: "Windows · macOS 13+",
    models: "Nemotron 3.5 Streaming and Parakeet TDT v3",
    purpose: "Live dictation with Nemotron; completed recordings and audio files with either model.",
    modelProfiles: ["nemotron-3.5-streaming", "parakeet-tdt-v3"],
    guide: "/docs/guides/local-runtime/#set-up-local-transcription",
  },
  {
    backend: "whisper-cpp",
    name: "whisper.cpp",
    platforms: "Windows only",
    models: "Whisper Base, Small, and Medium",
    purpose: "Completed recordings and audio files. Live dictation is not available.",
    modelProfiles: ["whisper-family"],
    guide: "/docs/guides/local-runtime/#whispercpp-transcription",
  },
  {
    backend: "llama-cpp",
    name: "llama.cpp",
    platforms: "Windows · macOS 13.3+",
    models: "S1-mini by Superwhisper",
    purpose: "English transcript cleanup after recognition, with raw text kept as the fallback.",
    modelProfiles: ["s1-mini"],
    guide: "/docs/guides/local-runtime/#local-cleanup-with-s1-mini",
  },
];
