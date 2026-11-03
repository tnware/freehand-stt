import {
  ProviderID,
  type Model,
  type ProviderDescriptor,
  type InstanceStatus,
} from "$bindings/managedruntime";
import { Role } from "$bindings/compatibility";
import { runtimeSidebarContext } from "./runtimeSidebar";
import { runtimePresentation } from "./managedRuntime";

/** Metadata-only guidance. Runtime setup still owns hardware checks and downloads. */
export function localSpeechSetup(
  providers: readonly ProviderDescriptor[],
  instances: readonly InstanceStatus[] = [],
) {
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
  const resumable = instances
    .flatMap((instance) => {
      const provider = available.find(
        (entry) => entry.id === instance.instance.provider,
      );
      const model = provider?.models?.find(
        (entry) => entry.id === instance.instance.model,
      );
      if (
        !provider ||
        !model ||
        !transcription(model) ||
        !instance.status.supported
      )
        return [];
      const view = runtimePresentation(instance.status);
      return [
        {
          provider,
          instance,
          context: runtimeSidebarContext(provider, instance),
          priority: view.ready ? 2 : view.installed ? 1 : 0,
        },
      ];
    })
    .toSorted(
      (a, b) =>
        b.priority - a.priority ||
        a.instance.instance.id.localeCompare(b.instance.instance.id),
    );
  const resumed = resumable[0];
  const existing = resumed
    ? {
        provider: resumed.provider,
        instance: resumed.instance,
        context: resumed.context,
      }
    : null;
  return {
    existing,
    availability:
      providers.length === 0
        ? "unknown"
        : available.length
          ? "available"
          : "unavailable",
    recommendation,
  } as const;
}
