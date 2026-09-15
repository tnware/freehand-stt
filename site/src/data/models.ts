// Editorial summaries of the dedicated profiles in internal/modelprofile.
// This directory describes Freehand controls, not the server's model inventory.
export const modelProfiles = [
  {
    id: "s1-mini",
    task: "Transcript cleanup",
    name: "S1-mini",
    family: "Superwhisper · Transcript cleanup",
    icon: "s1-mini",
    summary:
      "Format English transcripts with the style and structure you want.",
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
      "Choose style, structure, and context in Cleanup. Freehand requests reasoning off with llama.cpp and vLLM; with Generic, you must turn it off on the server.",
    note: "English cleanup after transcription. The raw transcript remains the fallback if cleanup fails.",
  },
  {
    id: "nemotron-3.5-streaming",
    task: "Transcription",
    name: "Nemotron 3.5 ASR",
    family: "NVIDIA · Streaming 0.6B",
    icon: "nemotron-3.5-streaming",
    summary:
      "Dictate live or transcribe recordings with your language and terminology.",
    guide: "/docs/models/nemotron/",
    backends: [
      { name: "NeMo-Speech.cpp", guide: "/docs/backends/nemo-speech/" },
    ],
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
    task: "Transcription",
    name: "Qwen3-ASR",
    family: "Qwen · Speech recognition",
    icon: "qwen3-asr",
    summary:
      "Use context for completed audio, or follow your dictation as live text.",
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
    task: "Transcription",
    name: "Parakeet TDT v3",
    family: "NVIDIA · Multilingual recognition",
    icon: "parakeet-tdt-v3",
    summary:
      "Transcribe recordings with automatic language detection and punctuation.",
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
      "Choose Parakeet TDT v3 in Voice or Audio-file transcription for completed recordings. Language detection is automatic; NeMo transcription controls cover punctuation and optional server formatting. Recognition hints and live mode are unavailable.",
    note: "This profile covers the TDT v3 checkpoint on NeMo-Speech.cpp.",
  },
  {
    id: "cohere-transcribe",
    task: "Transcription",
    name: "Cohere Transcribe",
    family: "Cohere · Speech recognition",
    icon: "cohere-transcribe",
    summary:
      "Transcribe completed audio with an explicit choice of fourteen languages.",
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
    note: "Cohere Transcribe on vLLM does not support context or vocabulary hints.",
  },
  {
    id: "voxtral-realtime",
    task: "Transcription",
    name: "Voxtral Mini Realtime",
    family: "Mistral · Streaming recognition",
    icon: "voxtral-realtime",
    summary:
      "Follow your dictation in the results pane and single-row overlay captions.",
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
      "Select Voxtral Mini Realtime and enable Realtime transcription in Voice’s Transcription panel. Stop recording to finalize the text and apply your usual cleanup and delivery settings.",
    note: "Uses the Voxtral Mini 4B Realtime checkpoint. Audio files retain their own selection.",
  },
  {
    id: "magpie-tts-multilingual-357m",
    task: "Text to speech",
    name: "MagpieTTS Multilingual 357M",
    family: "NVIDIA · v2602 speech generation",
    icon: "magpie-tts-multilingual-357m",
    summary:
      "Generate speech with preset voices alongside your local transcription model.",
    guide: "/docs/models/magpie-tts/",
    backends: [
      { name: "NeMo-Speech.cpp", guide: "/docs/backends/nemo-speech/" },
    ],
    controls: [
      { name: "Voice", detail: "Five preset speakers" },
      { name: "Language", detail: "Up to nine, depending on the server" },
      { name: "Local runtime", detail: "Shares NeMo with transcription" },
    ],
    inApp:
      "Get and select MagpieTTS in NeMo's catalog, then choose NeMo in Text to speech. Refresh voices reads the loaded speech model's metadata.",
    note: "Uses the v2602 checkpoint at normal speed. Japanese and Chinese require the corresponding server language frontends.",
  },
  {
    id: "qwen3-tts-customvoice",
    task: "Text to speech",
    name: "Qwen3-TTS",
    family: "Qwen · 1.7B CustomVoice",
    icon: "qwen3-tts-customvoice",
    summary:
      "Choose a preset speaker and describe how you want the words spoken.",
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
    note: "Requires the 1.7B CustomVoice checkpoint. Voice cloning and VoiceDesign are not supported by this profile.",
  },
] as const;
