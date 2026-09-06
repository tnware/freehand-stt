import { vi } from "vitest";

vi.mock("$bindings/connection/service", () => ({}));
vi.mock("$bindings/dictation/service", () => ({}));
vi.mock("$bindings/filetranscription/service", () => ({}));
vi.mock("$bindings/history/service", () => ({}));
vi.mock("$bindings/input/service", () => ({}));
vi.mock("$bindings/settings/service", () => ({}));
vi.mock("$bindings/tts/service", () => ({
  CurrentStatus: vi.fn(),
  PlayHistoryEntry: vi.fn(),
  PlayFileTranscript: vi.fn(),
  PreviewVoice: vi.fn(),
  SpeakText: vi.fn(),
  Pause: vi.fn(),
  Resume: vi.fn(),
  Restart: vi.fn(),
  Stop: vi.fn(),
  SaveAudio: vi.fn(),
  ClearAudio: vi.fn(),
}));

export * from "./session-fixtures-data";
