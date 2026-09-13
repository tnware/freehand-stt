import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import test from "node:test";

const names = ["freehand-windows-amd64.exe", "freehand-windows-amd64-installer.exe", "freehand-darwin-arm64.zip", "freehand-darwin-amd64.zip", "SHA256SUMS"];
const payload = (names) => ({ assets: names.map((name) => ({ name })) });
const run = (input) => spawnSync(process.execPath, [new URL("./check.mjs", import.meta.url).pathname], { input, encoding: "utf8" });
test("accepts exactly the public asset names in any order", () => {
  const result = run(JSON.stringify(payload([...names].reverse())));
  assert.equal(result.status, 0, result.stderr);
});
for (const name of names) {
  test(`rejects missing ${name}`, () => assert.notEqual(run(JSON.stringify(payload(names.filter((n) => n !== name)))).status, 0));
}
for (const [label, value] of Object.entries({
  unexpected: payload([...names, "old.zip"]),
  sidecar: payload([...names, "freehand.exe.sha256"]),
  duplicate: payload([...names, names[0]]),
  replacement: payload([...names.slice(1), names[1]]),
  absent: {},
  null: null,
  wrongType: { assets: {} },
  malformedEntry: { assets: [...payload(names).assets, null] },
  malformedName: payload([...names.slice(1), 123]),
  path: payload([...names.slice(1), `../${names[0]}`]),
  whitespace: payload([...names.slice(1), `${names[0]}\n`]),
})) {
  test(`rejects ${label}`, () => assert.notEqual(run(JSON.stringify(value)).status, 0));
}
test("rejects invalid JSON", () => assert.notEqual(run("not json").status, 0));
