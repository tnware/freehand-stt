import { ProviderID, type Model } from "$bindings/managedruntime";

export type RuntimeModelFamily = { id: string; label: string; models: Model[] };

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
