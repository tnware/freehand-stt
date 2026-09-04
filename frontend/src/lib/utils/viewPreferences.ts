/**
 * Disclosure state is renderer-only presentation state. It stays out of the
 * durable application settings while remaining stable across window launches.
 */
export type HomeDisclosure = "history" | "quick-stt" | "quick-cleanup";

type PreferenceStorage = Pick<Storage, "getItem" | "setItem">;

const disclosureKeys: Record<HomeDisclosure, string> = {
  history: "freehand:view:v1:history-open",
  "quick-stt": "freehand:view:v1:quick-stt-open",
  "quick-cleanup": "freehand:view:v1:quick-cleanup-open",
};

function rendererStorage(): PreferenceStorage | undefined {
  try {
    return globalThis.localStorage;
  } catch {
    return undefined;
  }
}

export function readDisclosurePreference(
  disclosure: HomeDisclosure,
  fallback: boolean,
  storage: PreferenceStorage | undefined = rendererStorage(),
): boolean {
  if (!storage) return fallback;
  try {
    const value = storage.getItem(disclosureKeys[disclosure]);
    if (value === "open") return true;
    if (value === "closed") return false;
  } catch {
    // WebView storage is a convenience, not an application-state dependency.
  }
  return fallback;
}

export function writeDisclosurePreference(
  disclosure: HomeDisclosure,
  open: boolean,
  storage: PreferenceStorage | undefined = rendererStorage(),
): void {
  if (!storage) return;
  try {
    storage.setItem(disclosureKeys[disclosure], open ? "open" : "closed");
  } catch {
    // Keep the disclosure usable when storage is unavailable or quota-limited.
  }
}
