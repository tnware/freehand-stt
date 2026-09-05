import catalog from "./compatibility.generated.json";

// Editorial copy and guide destinations live here. Availability and capabilities
// come only from the generated Go catalog; never duplicate support flags here.
const editorial = [
  { id: "generic", short: "API", summary: "Start with the common contract. Connect an endpoint you choose for transcription, cleanup, or speech playback.", guide: "/docs/backends/generic/", evidence: "Automated contract fixtures", detail: "A protocol baseline, usable with servers that implement the documented request and response shapes." },
  { id: "speaches", short: "SP", summary: "Use one speech server for dictation, audio files, and on-demand voice playback.", guide: "/docs/backends/speaches/", evidence: "Reported working setup + contract fixtures", detail: "Whisper-family transcription and Kokoro speech are reported working. Support varies by model and server version." },
  { id: "llama-cpp", short: "LC", summary: "Clean up a completed transcript with a separately hosted text model, including the optional S1-mini preset.", guide: "/docs/backends/llama-cpp/", evidence: "Reported working setup + contract fixtures", detail: "S1-mini cleanup is reported working. Other text models must accept the documented chat contract." },
  { id: "openai", short: "OA", summary: "A dedicated profile for OpenAI's hosted service and model-specific rules.", guide: "/docs/backends/planned/#openai-hosted", evidence: "Dedicated profile planned", detail: "Qualify model-specific fields, limits, and response formats." },
  { id: "localai", short: "LA", summary: "An operation-aware profile for LocalAI's different inference backends.", guide: "/docs/backends/planned/#localai", evidence: "Dedicated profile planned", detail: "Qualify the selected backend's request fields, streaming, and audio output." },
  { id: "whisper-cpp", short: "WC", summary: "A transcription profile for the native whisper.cpp HTTP server.", guide: "/docs/backends/planned/#whispercpp", evidence: "Dedicated profile planned", detail: "Handle native routing and the server-loaded model explicitly." },
  { id: "vllm", short: "VL", summary: "Dedicated transcription and text-cleanup contracts for vLLM.", guide: "/docs/backends/planned/#vllm", evidence: "Dedicated profile planned", detail: "Implement its transcription stream decoder and qualify text-processing behavior." },
  { id: "vllm-omni", short: "VO", summary: "A speech playback profile with explicit model and voice requirements.", guide: "/docs/backends/planned/#vllm-omni", evidence: "Dedicated profile planned", detail: "Qualify preset-voice inputs and playable WAV output separately from cloning features." },
  { id: "kokoro-fastapi", short: "KF", summary: "A dedicated connection contract for Kokoro-FastAPI speech generation.", guide: "/docs/backends/planned/#kokoro-fastapi", evidence: "Dedicated profile planned", detail: "Qualify voice IDs, speed handling, and compatible WAV responses." },
  { id: "openedai-speech", short: "OS", summary: "A speech profile for endpoints with server-configured voice aliases.", guide: "/docs/backends/planned/#openedai-speech", evidence: "Dedicated profile planned", detail: "Qualify the voice configuration and the returned audio encoding." },
];

export const roles = [
  { key: "transcription", label: "Speech to text" },
  { key: "postProcessing", label: "Transcript cleanup" },
  { key: "speech", label: "Speech playback" },
] as const;
export type Support = "available" | "planned" | "none";
export const supportLabel: Record<Support, string> = { available: "Supported", planned: "Planned", none: "—" };
export const features = [
  { key: "microphone", label: "Dictation" },
  { key: "files", label: "Audio files" },
  { key: "streaming", label: "File streaming" },
  { key: "prompt", label: "STT context" },
  { key: "hotwords", label: "STT hotwords" },
  { key: "temperature", label: "STT temperature" },
  { key: "cleanup", label: "Cleanup" },
  { key: "playback", label: "Speech playback" },
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
  const status = (entry: typeof stt): Support => entry ? (entry.available ? "available" : "planned") : "none";
  return {
    ...copy,
    name: entries[0].label,
    available: entries.some((entry) => entry.available),
    entries,
    features: {
      microphone: status(stt),
      files: status(stt),
      streaming: stt?.available ? (stt.capabilities.fileStreaming ? "available" : "none") : "none",
      prompt: stt?.available && stt.capabilities.transcriptionPrompt ? "available" : "none",
      hotwords: stt?.available && stt.capabilities.transcriptionHotwords ? "available" : "none",
      temperature: stt?.available && stt.capabilities.transcriptionTemperature ? "available" : "none",
      cleanup: status(chat),
      playback: status(speech),
    } satisfies Record<(typeof features)[number]["key"], Support>,
  };
});
export const availableBackends = backends.filter((entry) => entry.available);
export const plannedBackends = backends.filter((entry) => !entry.available);
