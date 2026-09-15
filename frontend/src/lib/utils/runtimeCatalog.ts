import { ProviderID, type Model } from "$bindings/managedruntime";
import { Role } from "$bindings/compatibility";

export type RuntimeModelFamily = { id: string; label: string; models: Model[] };
export type RuntimeModelGroup = {
  id: string;
  label: string;
  families: RuntimeModelFamily[];
};

/** Every model gets one group based on qualified task contracts, never its ID. */
export function runtimeModelGroups(
  provider: ProviderID,
  models: readonly Model[],
): RuntimeModelGroup[] {
  const tasks = [
    {
      id: "transcription",
      label: "Transcription",
      roles: [Role.Transcription, Role.Realtime],
    },
    { id: "speech", label: "Text to speech", roles: [Role.Speech] },
    { id: "cleanup", label: "Cleanup", roles: [Role.PostProcessing] },
  ];
  const buckets = new Map<string, Model[]>();
  for (const model of models) {
    const id =
      tasks.find((task) =>
        model.contracts?.some((contract) => task.roles.includes(contract.role)),
      )?.id ?? "other";
    const bucket = buckets.get(id) ?? [];
    bucket.push(model);
    buckets.set(id, bucket);
  }
  return [...tasks, { id: "other", label: "Other models", roles: [] }].flatMap(
    (task) => {
      const entries = buckets.get(task.id);
      if (!entries?.length) return [];
      return [
        {
          id: task.id,
          label: buckets.size > 1 ? task.label : "",
          families: runtimeModelFamilies(provider, entries).map((family) => ({
            ...family,
            id: `${task.id}:${family.id}`,
          })),
        },
      ];
    },
  );
}

function whisperFamily(id: string) {
  return (
    /^(tiny|base|small|medium)(?:\.en)?(?:-q[58]_[01])?$/.exec(id)?.[1] ??
    (/^large-v[123](?:-turbo)?(?:-q[58]_[01])?$/.test(id) ? "large" : undefined)
  );
}

export function runtimeVariantLabel(provider: ProviderID, id: string): string {
  if (provider !== ProviderID.WhisperCPP || !whisperFamily(id)) return "";
  const language = id.includes(".en") ? "English only" : "Multilingual";
  const quantization =
    /-(q[58]_[01])$/.exec(id)?.[1].toUpperCase() ?? "Standard";
  return `${language} · ${quantization}`;
}

/** Presentation only: callers supply the already-qualified provider catalog. */
export function runtimeModelFamilies(
  provider: ProviderID,
  models: readonly Model[],
): RuntimeModelFamily[] {
  if (provider !== ProviderID.WhisperCPP)
    return [{ id: "all", label: "", models: [...models] }];

  const families = ["tiny", "base", "small", "medium", "large"] as const;
  const grouped = new Map<string, Model[]>();
  for (const model of models) {
    // These anchored model families only organize names. They never assign
    // capabilities, profiles, downloads, or runtime settings.
    const family = whisperFamily(model.id) ?? "other";
    const entries = grouped.get(family) ?? [];
    entries.push(model);
    grouped.set(family, entries);
  }
  if (grouped.size === 1 && grouped.has("other"))
    return [{ id: "all", label: "", models: [...models] }];
  return [...families, "other"].flatMap((id) => {
    const entries = grouped.get(id);
    return entries?.length
      ? [
          {
            id,
            label:
              id === "other"
                ? "Other models"
                : id[0].toUpperCase() + id.slice(1),
            models: entries,
          },
        ]
      : [];
  });
}
