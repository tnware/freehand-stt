import { describe, expect, it } from "vitest";
import { readDisclosurePreference, writeDisclosurePreference } from "./viewPreferences";

function memoryStorage(initial: Record<string, string> = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem(key: string) {
      return values.get(key) ?? null;
    },
    setItem(key: string, value: string) {
      values.set(key, value);
    },
  };
}

describe("home view preferences", () => {
  it("uses the disclosure's safe default until a preference is stored", () => {
    const storage = memoryStorage();

    expect(readDisclosurePreference("history", true, storage)).toBe(true);
    expect(readDisclosurePreference("history", false, storage)).toBe(false);
  });

  it("persists an explicit choice over the default", () => {
    const storage = memoryStorage();

    writeDisclosurePreference("history", false, storage);
    expect(readDisclosurePreference("history", true, storage)).toBe(false);

    writeDisclosurePreference("history", true, storage);
    expect(readDisclosurePreference("history", false, storage)).toBe(true);

    writeDisclosurePreference("quick-stt", false, storage);
    writeDisclosurePreference("quick-cleanup", true, storage);
    expect(readDisclosurePreference("quick-stt", true, storage)).toBe(false);
    expect(readDisclosurePreference("quick-cleanup", false, storage)).toBe(true);
  });

  it("falls back when WebView storage is unavailable or malformed", () => {
    const unavailable = {
      getItem() {
        throw new Error("storage unavailable");
      },
      setItem() {
        throw new Error("storage unavailable");
      },
    };
    const malformed = memoryStorage({ "freehand:view:v1:history-open": "sometimes" });

    expect(readDisclosurePreference("history", true, unavailable)).toBe(true);
    expect(() => writeDisclosurePreference("history", false, unavailable)).not.toThrow();
    expect(readDisclosurePreference("history", false, malformed)).toBe(false);
  });
});
