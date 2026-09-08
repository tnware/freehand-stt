// Editorial summaries of the dedicated profiles in internal/modelprofile.
// This directory describes Freehand controls, not the server's model inventory.
export const modelProfiles = [
  {
    id: "s1-mini",
    name: "S1-mini",
    family: "Superwhisper · Transcript cleanup",
    icon: null,
    summary: "Turn spoken English into text with the style and structure you want.",
    guide: "/docs/models/s1-mini/",
    backends: [
      { name: "llama.cpp", guide: "/docs/backends/llama-cpp/" },
      { name: "vLLM", guide: "/docs/backends/vllm/" },
      { name: "Generic chat", guide: "/docs/backends/generic/" },
    ],
    controls: [
      { name: "Styling", detail: "Casual through formal" },
      { name: "Structure", detail: "Prose or lists" },
      { name: "Context", detail: "General or email" },
    ],
    inApp:
      "The Cleanup panel shows trained output controls and builds the model's fixed prompt for you. llama.cpp and vLLM requests enforce reasoning off; Generic requires it on the server.",
    note: "English cleanup after transcription. The raw transcript remains the fallback if cleanup fails.",
  },
  {
    id: "nemotron-3.5-streaming",
    name: "Nemotron 3.5 ASR",
    family: "NVIDIA · Streaming 0.6B",
    icon: "nemo-speech-v1",
    summary: "Dictate live or transcribe recordings with your language and terminology.",
    guide: "/docs/models/nemotron/",
    backends: [{ name: "NeMo-Speech.cpp", guide: "/docs/backends/nemo-speech/" }],
    controls: [
      { name: "Language", detail: "Automatic or 32 locales" },
      { name: "Vocabulary", detail: "Shared terms and strength" },
      { name: "Live mode", detail: "Results and optional captions" },
    ],
    inApp:
      "The Transcription panel shows Nemotron's languages and vocabulary controls. In Voice, the Realtime transcription toggle switches the same connection and model into live mode.",
    note: "Language and vocabulary work in completed and live modes. Audio files keep their own settings.",
  },
  {
    id: "qwen3-asr",
    name: "Qwen3-ASR",
    family: "Qwen · Speech recognition",
    icon: "qwen3-asr",
    summary: "Use context for completed audio, or follow your dictation as live text.",
    guide: "/docs/models/qwen3-asr/",
    backends: [{ name: "vLLM", guide: "/docs/backends/vllm/" }],
    controls: [
      { name: "Completed audio", detail: "Language, context, temperature" },
      { name: "Vocabulary", detail: "Shared terms in completed context" },
      { name: "Live mode", detail: "Automatic language and captions" },
    ],
    inApp:
      "Choose Qwen3-ASR in the Transcription panel to see its recognition options. Enabling realtime shows live results and optional captions while preserving completed-mode settings.",
    note: "The realtime setup uses the 1.7B checkpoint with vLLM's realtime architecture. Context and vocabulary apply to completed audio only.",
  },
] as const;
