import manifest from "./manifest.json";
import generic from "./generic.svg?url";
import speaches from "./speaches.svg?url";
import whisper from "./whisper-cpp.svg?url";
import llama from "./llama-cpp.svg?url";
import vllm from "./vllm.svg?url";
import openai from "./openai.svg?url";
import localai from "./localai.svg?url";
import speech from "./speech.svg?url";

const assets: Record<string, string> = {
  "generic.svg": generic, "speaches.svg": speaches, "whisper-cpp.svg": whisper,
  "llama-cpp.svg": llama, "vllm.svg": vllm, "openai.svg": openai,
  "localai.svg": localai, "speech.svg": speech,
};

/** Presentation only. Capability and availability decisions belong to the Go catalog. */
export function providerIdentity(id: string | null | undefined) {
  const entry = manifest[Object.hasOwn(manifest, id ?? "") ? id as keyof typeof manifest : "generic"];
  return { ...entry, src: assets[entry.asset] ?? generic };
}
