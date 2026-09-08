// Editorial summaries of the dedicated profiles in internal/modelprofile.
// This directory describes Freehand controls, not the server's model inventory.
export const modelProfiles = [
  {
    id: "s1-mini",
    name: "S1-mini",
    family: "Superwhisper · Transcript cleanup",
    icon: "s1-mini",
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
    icon: "nemotron-3.5-streaming",
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
  {
    id: "parakeet-tdt-v3",
    name: "Parakeet TDT v3",
    family: "NVIDIA · Multilingual recognition",
    icon: "parakeet-tdt-v3",
    summary: "Transcribe recordings with automatic language detection and punctuation.",
    guide: "/docs/models/parakeet/",
    backends: [
      {
        name: "NeMo-Speech.cpp",
        guide: "/docs/backends/nemo-speech/",
      },
    ],
    controls: [
      {
        name: "Recognition",
        detail: "Automatic language and punctuation",
      },
      {
        name: "Voice",
        detail: "Recordings and checkpoints",
      },
      {
        name: "Audio files",
        detail: "Completed results",
      },
    ],
    inApp:
      "Choose Parakeet TDT v3 in Voice or Audio-file transcription. Its profile keeps the completed workflow and hides unsupported recognition hints and realtime controls.",
    note: "This profile covers the TDT v3 checkpoint on NeMo-Speech.cpp.",
  },
  {
    id: "cohere-transcribe",
    name: "Cohere Transcribe",
    family: "Cohere · Speech recognition",
    icon: "cohere-transcribe",
    summary: "Transcribe completed audio with an explicit choice of fourteen languages.",
    guide: "/docs/models/cohere-transcribe/",
    backends: [
      {
        name: "vLLM",
        guide: "/docs/backends/vllm/",
      },
    ],
    controls: [
      {
        name: "Language",
        detail: "14 languages; English default",
      },
      {
        name: "Recognition",
        detail: "Temperature override",
      },
      {
        name: "Audio files",
        detail: "Optional streamed results",
      },
    ],
    inApp:
      "The Transcription panel shows Cohere’s language choices and temperature control. Punctuation follows vLLM’s model adapter.",
    note: "Context and vocabulary controls are absent because this adapter does not apply them.",
  },
  {
    id: "voxtral-realtime",
    name: "Voxtral Mini Realtime",
    family: "Mistral · Streaming recognition",
    icon: "voxtral-realtime",
    summary: "Follow your dictation in the results pane and single-row overlay captions.",
    guide: "/docs/models/voxtral-realtime/",
    backends: [
      {
        name: "vLLM",
        guide: "/docs/backends/vllm/",
      },
    ],
    controls: [
      {
        name: "Live mode",
        detail: "Results and optional captions",
      },
      {
        name: "Language",
        detail: "Automatic detection",
      },
      {
        name: "Completed audio",
        detail: "Recordings and files",
      },
    ],
    inApp:
      "Select Voxtral Mini Realtime and enable Realtime transcription within Voice’s existing Transcription panel. Stop to finalize and apply your usual cleanup and delivery.",
    note: "Uses the Voxtral Mini 4B Realtime checkpoint. Audio files retain their own selection.",
  },
  {
    id: "qwen3-tts-customvoice",
    name: "Qwen3-TTS",
    family: "Qwen · 1.7B CustomVoice",
    icon: "qwen3-tts-customvoice",
    summary: "Choose a preset speaker and describe how you want the words spoken.",
    guide: "/docs/models/qwen3-tts/",
    backends: [
      {
        name: "vLLM-Omni",
        guide: "/docs/backends/vllm-omni/",
      },
    ],
    controls: [
      {
        name: "Voice",
        detail: "Nine preset speakers",
      },
      {
        name: "Language",
        detail: "Automatic or ten languages",
      },
      {
        name: "Voice style",
        detail: "Natural-language instructions",
      },
    ],
    inApp:
      "Text to speech and its quick settings show language and voice-style controls. Preview uses your current edits before saving, including the selected voice and speaking speed.",
    note: "CustomVoice 1.7B behavior. Voice cloning and VoiceDesign use different model contracts.",
  },
] as const;
