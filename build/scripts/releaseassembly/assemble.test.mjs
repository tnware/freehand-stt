import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const assets = [
  ["freehand.exe", "freehand-windows-amd64.exe"],
  ["freehand-amd64-installer.exe", "freehand-windows-amd64-installer.exe"],
  ["freehand-darwin-arm64.zip", "freehand-darwin-arm64.zip"],
  ["freehand-darwin-amd64.zip", "freehand-darwin-amd64.zip"],
];
function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), "freehand-release-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const input = join(root, "bin"), output = join(root, "dist");
  mkdirSync(input);
  for (const [source] of assets) writeFileSync(join(input, source), `fixture:${source}`);
  const run = () => spawnSync(process.execPath, [fileURLToPath(new URL("./assemble.mjs", import.meta.url)), input, output], { encoding: "utf8" });
  return { input, output, run };
}
test("assembles exactly four byte-preserved assets and one complete checksum manifest", (t) => {
  const { input, output, run } = fixture(t);
  // Valid CI checksum sidecars must not leak into the exact public set.
  writeFileSync(join(input, "freehand.exe.sha256"), "CI checksum sidecar");
  const result = run();
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(readdirSync(output).sort(), ["SHA256SUMS", ...assets.map(([, name]) => name)].sort());
  const lines = [];
  for (const [source, name] of assets) {
    const bytes = readFileSync(join(input, source));
    assert.deepEqual(readFileSync(join(output, name)), bytes);
    lines.push(`${createHash("sha256").update(bytes).digest("hex")}  ${name}`);
  }
  assert.equal(readFileSync(join(output, "SHA256SUMS"), "utf8"), lines.sort((a, b) => a.split("  ")[1].localeCompare(b.split("  ")[1])).join("\n") + "\n");
});
for (const [source] of assets) {
  for (const state of ["missing", "empty", "directory"]) {
    test(`rejects ${state} ${source} without leaving publishable output`, (t) => {
      const { input, output, run } = fixture(t);
      rmSync(join(input, source));
      if (state === "empty") writeFileSync(join(input, source), "");
      if (state === "directory") mkdirSync(join(input, source));
      assert.notEqual(run().status, 0);
      assert.equal(existsSync(output), false);
    });
  }
}
