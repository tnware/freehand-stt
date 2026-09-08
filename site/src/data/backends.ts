import catalog from "./compatibility.generated.json";

// Editorial copy and guide destinations live here. Availability and capabilities
// come only from the generated Go catalog; never duplicate support flags here.
const editorial = [
  {
    id: "nemo-speech-v1",
    summary: "Transcribe recordings or follow live dictation with Nemotron.",
    guide: "/docs/backends/nemo-speech/",
    highlight: "Live text and captions",
    detail:
      "Choose the Nemotron model profile for realtime transcription, language options, and vocabulary hints.",
  },
  {
    id: "generic",
    summary: "Connect OpenAI-compatible services for transcription, cleanup, or text to speech.",
    guide: "/docs/backends/generic/",
    highlight: "Bring your existing endpoint",
    detail: "Choose the operations your service offers, then enter its URL and model ID.",
  },
  {
    id: "speaches",
    summary: "Use one speech server for dictation, audio files, and text to speech.",
    guide: "/docs/backends/speaches/",
    highlight: "Speech in both directions",
    detail:
      "Connect Whisper-family transcription and Kokoro speech models, with searchable voices and streaming file results.",
  },
  {
    id: "llama-cpp",
    summary: "Polish completed transcripts with a text model running in llama.cpp.",
    guide: "/docs/backends/llama-cpp/",
    highlight: "Cleanup that fits your writing",
    detail:
      "Use your own instructions or the S1-mini preset for punctuation, formatting, and style.",
  },
  {
    id: "openai",
    summary: "A dedicated profile for OpenAI's hosted service and model-specific rules.",
    guide: "/docs/backends/planned/#openai-hosted",
    highlight: "Planned integration",
    detail: "Qualify model-specific fields, limits, and response formats.",
  },
  {
    id: "localai",
    summary: "An operation-aware profile for LocalAI's different inference backends.",
    guide: "/docs/backends/planned/#localai",
    highlight: "Planned integration",
    detail: "Qualify the selected backend's request fields, streaming, and audio output.",
  },
  {
    id: "whisper-cpp",
    summary: "Transcribe microphone recordings and audio files with whisper.cpp.",
    guide: "/docs/backends/whisper-cpp/",
    highlight: "Use the model already loaded",
    detail: "Connect to the server and start transcribing. Model selection stays in whisper.cpp.",
  },
  {
    id: "vllm",
    summary: "Connect speech recognition and text cleanup models through vLLM.",
    guide: "/docs/backends/vllm/",
    highlight: "Qwen3-ASR realtime and more",
    detail:
      "Use the Qwen3-ASR model profile for live dictation, or choose completed transcription, streaming file results, and optional cleanup.",
  },
  {
    id: "vllm-omni",
    summary: "A speech playback profile with explicit model and voice requirements.",
    guide: "/docs/backends/planned/#vllm-omni",
    highlight: "Planned integration",
    detail: "Qualify preset-voice inputs and playable WAV output separately from cloning features.",
  },
  {
    id: "kokoro-fastapi",
    summary: "Generate spoken audio with Kokoro-FastAPI.",
    guide: "/docs/backends/kokoro-fastapi/",
    highlight: "Find the voice you want",
    detail:
      "Browse available voices, enter custom voice IDs, and adjust speaking speed. Listen in Freehand or save the generated WAV.",
  },
  {
    id: "openedai-speech",
    summary: "A speech profile for endpoints with server-configured voice aliases.",
    guide: "/docs/backends/planned/#openedai-speech",
    highlight: "Planned integration",
    detail: "Qualify the voice configuration and the returned audio encoding.",
  },
];

export const roles = [
  { key: "transcription", label: "Speech to text" },
  { key: "postProcessing", label: "Transcript cleanup" },
  { key: "speech", label: "Speech playback" },
] as const;
export type Support = "available" | "planned" | "none";
export const supportLabel: Record<Support, string> = {
  available: "Supported",
  planned: "Planned",
  none: "—",
};
export const features = [
  { key: "microphone", label: "Dictation" },
  { key: "files", label: "Audio files" },
  { key: "streaming", label: "File streaming" },
  { key: "language", label: "Language selection" },
  { key: "prompt", label: "STT context" },
  { key: "hotwords", label: "STT hotwords" },
  { key: "temperature", label: "STT temperature" },
  { key: "cleanup", label: "Cleanup" },
  { key: "cleanupLimit", label: "Cleanup token limit" },
  { key: "reasoningOff", label: "Disable reasoning" },
  { key: "playback", label: "Speech playback" },
  { key: "voices", label: "Voice discovery" },
] as const;
const descriptions = new Map(editorial.map((entry) => [entry.id, entry]));
const ids = [...new Set(roles.flatMap(({ key }) => catalog[key].map((profile) => profile.id)))];
if (editorial.length !== ids.length || editorial.some((entry) => !ids.includes(entry.id))) {
  throw new Error("Backend guide directory must cover every app profile exactly once.");
}

export const backends = ids.map((id) => {
  const copy = descriptions.get(id);
  if (!copy) throw new Error(`Missing backend guide: ${id}`);
  const stt = catalog.transcription.find((entry) => entry.id === id);
  const chat = catalog.postProcessing.find((entry) => entry.id === id);
  const speech = catalog.speech.find((entry) => entry.id === id);
  const entries = roles.flatMap(({ key, label }) => {
    const profile = catalog[key].find((entry) => entry.id === id);
    return profile ? [{ ...profile, role: label }] : [];
  });
  const status = (entry: typeof stt): Support =>
    entry ? (entry.available ? "available" : "planned") : "none";
  return {
    ...copy,
    name: entries[0].label,
    available: entries.some((entry) => entry.available),
    entries,
    features: {
      microphone: status(stt),
      files: status(stt),
      streaming: stt?.available ? (stt.capabilities.fileStreaming ? "available" : "none") : "none",
      language: stt?.available && stt.capabilities.languageHint ? "available" : "none",
      prompt: stt?.available && stt.capabilities.transcriptionPrompt ? "available" : "none",
      hotwords: stt?.available && stt.capabilities.transcriptionHotwords ? "available" : "none",
      temperature:
        stt?.available && stt.capabilities.transcriptionTemperature ? "available" : "none",
      cleanup: status(chat),
      cleanupLimit: chat?.available && chat.capabilities.cleanupOutputLimit ? "available" : "none",
      reasoningOff:
        chat?.available && chat.capabilities.cleanupDisableReasoning ? "available" : "none",
      playback: status(speech),
      voices: speech?.available && speech.capabilities.voiceDiscovery ? "available" : "none",
    } satisfies Record<(typeof features)[number]["key"], Support>,
  };
});
export const availableBackends = backends.filter((entry) => entry.available);
export const plannedBackends = backends.filter((entry) => !entry.available);
