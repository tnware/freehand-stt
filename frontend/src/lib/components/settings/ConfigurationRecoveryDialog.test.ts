import { readFileSync } from "node:fs";
import { expect, it } from "vitest";

it("limits recovery guidance to the current database and explicit reconfiguration", () => {
  const source = readFileSync(
    new URL("./ConfigurationRecoveryDialog.svelte", import.meta.url),
    "utf8",
  ).replace(/\s+/g, " ");

  expect(source).not.toContain("legacy import");
  expect(source).toContain("current-version database backup");
  expect(source).toContain("Retry loading after fixing file access.");
  expect(source).toContain(
    "Reconfigure connections and enter API keys again after resetting.",
  );
  expect(source).toContain(
    "credentials stored in {native.credentialStore} are not deleted",
  );
});
