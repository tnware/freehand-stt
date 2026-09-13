// Editorial families using existing Generic model behavior, not additional app
// profiles or guarantees about every checkpoint a server might advertise.
export const modelFamilies = [
  {
    id: "whisper-family",
    task: "Transcription",
    name: "Whisper",
    family: "Whisper & Distil-Whisper · Completed transcription",
    icon: "whisper-cpp", // Existing neutral waveform, not an OpenAI brand claim.
    behavior: "Generic model behavior",
    controls: [
      { name: "Recognition", detail: "Language, context, and temperature where supported" },
      { name: "Vocabulary", detail: "Speaches hotwords; context hints on whisper.cpp" },
      { name: "Audio files", detail: "Completed results; optional streaming on Speaches" },
    ],
    note: "Choose Generic model behavior with the appropriate backend profile. Checkpoint and language support belong to the server. whisper.cpp uses its server-loaded model and does not support file streaming or live dictation in Freehand.",
    guide: "/docs/models/families/#whisper",
    backends: [
      { name: "Speaches", guide: "/docs/backends/speaches/" },
      { name: "whisper.cpp", guide: "/docs/backends/whisper-cpp/" },
      { name: "Compatible STT", guide: "/docs/backends/generic/" },
    ],
  },
  {
    id: "generic-chat-models",
    task: "Transcript cleanup",
    name: "Compatible chat models",
    family: "Instruction-following text models · Optional cleanup",
    icon: "generic",
    behavior: "Generic model behavior",
    controls: [
      { name: "Instructions", detail: "Custom cleanup instructions" },
      { name: "Output", detail: "Generation controls provided by the backend" },
      { name: "Recovery", detail: "Raw transcript on cleanup failure" },
    ],
    note: "Use an instruction-following model behind a compatible chat API. Generic does not apply S1-mini’s trained style controls or establish compatibility for every advertised model.",
    guide: "/docs/guides/post-processing/",
    backends: [
      { name: "llama.cpp", guide: "/docs/backends/llama-cpp/" },
      { name: "vLLM", guide: "/docs/backends/vllm/" },
      { name: "Generic chat", guide: "/docs/backends/generic/" },
    ],
  },
  {
    id: "kokoro-family",
    task: "Text to speech",
    name: "Kokoro",
    family: "Speech generation · Preset voices",
    icon: "kokoro-fastapi", // Existing neutral speech symbol.
    behavior: "Generic model behavior",
    controls: [
      { name: "Voice", detail: "Browse available voices or enter an ID" },
      { name: "Speaking speed", detail: "Adjust speed before generation" },
      { name: "Audio", detail: "Listen or explicitly save the generated WAV" },
    ],
    note: "Choose Generic model behavior with Speaches or Kokoro-FastAPI. Voice availability depends on the server and model. Language overrides, voice-style instructions, and cloning are not exposed by these profiles.",
    guide: "/docs/backends/kokoro-fastapi/",
    backends: [
      { name: "Speaches", guide: "/docs/backends/speaches/" },
      { name: "Kokoro-FastAPI", guide: "/docs/backends/kokoro-fastapi/" },
    ],
  },
] as const;
