import {
  ProviderID,
  type Model,
  type ProviderDescriptor,
} from "$bindings/managedruntime";
import { Role } from "$bindings/compatibility";

/** Metadata-only guidance. Runtime setup still owns hardware checks and downloads. */
export function localSpeechSetup(providers: readonly ProviderDescriptor[]) {
  const transcription = (model: Model) =>
    model.contracts?.some((contract) => contract.role === Role.Transcription);
  const available = providers.filter(
    (provider) => provider.supported && provider.models?.some(transcription),
  );
  let recommendation: { provider: ProviderDescriptor; model: Model } | null =
    null;
  for (const id of [ProviderID.NeMoSpeechCPP, ProviderID.WhisperCPP]) {
    const provider = available.find((entry) => entry.id === id);
    const model = provider?.models?.find(
      (entry) => entry.recommended && transcription(entry),
    );
    if (provider && model) {
      recommendation = { provider, model };
      break;
    }
  }
  return {
    availability:
      providers.length === 0
        ? "unknown"
        : available.length
          ? "available"
          : "unavailable",
    recommendation,
  } as const;
}
