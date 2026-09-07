import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const script = fileURLToPath(new URL("./validation.mjs", import.meta.url));

function git(cwd, ...args) {
  const result = spawnSync("git", args, { cwd, encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim();
}

function fixture(t) {
  const cwd = mkdtempSync(join(tmpdir(), "freehand-ci-"));
  t.after(() => rmSync(cwd, { recursive: true, force: true }));
  git(cwd, "init", "--quiet");
  writeFileSync(join(cwd, "README.md"), "Before\n");
  git(cwd, "add", ".");
  git(cwd, "-c", "user.name=CI test", "-c", "user.email=ci@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Base");
  return { cwd, base: git(cwd, "rev-parse", "HEAD") };
}

function runChanges(cwd, base, extra = {}) {
  return spawnSync(process.execPath, [script, "changes"], {
    cwd,
    encoding: "utf8",
    env: { ...process.env, CI_BASE_SHA: base, CI_FORCE_FULL: "false", GITHUB_OUTPUT: join(cwd, "outputs"), ...extra },
  });
}

function commit(cwd) {
  git(cwd, "add", ".");
  git(cwd, "-c", "user.name=CI test", "-c", "user.email=ci@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "-m", "Change");
}

test("README-only changes skip application and site workloads", (t) => {
  const { cwd, base } = fixture(t);
  writeFileSync(join(cwd, "README.md"), "After\n");
  git(cwd, "add", ".");
  git(cwd, "-c", "user.name=CI test", "-c", "user.email=ci@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Docs");
  const result = runChanges(cwd, base);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(readFileSync(join(cwd, "outputs"), "utf8"), "app=false\nsite=false\n");
});

for (const [path, app, site] of [
  ["site/src/pages/index.astro", false, true],
  ["branding/README.md", false, false],
  ["frontend/src/App.svelte", true, false],
  ["internal/compatibility/catalog.go", true, true],
  ["build/scripts/compatibility/main.go", true, true],
  ["branding/freehand-mark.svg", true, true],
  ["branding/providers/provider.svg", true, true],
  ["go.mod", true, true],
  ["go.sum", true, true],
  [".github/workflows/ci.yml", true, true],
  [".github/actions/setup-go/action.yml", true, true],
  ["new/unknown-input.md", true, false],
]) {
  test(`selects relevant workloads for ${path}`, (t) => {
    const { cwd, base } = fixture(t);
    mkdirSync(dirname(join(cwd, path)), { recursive: true });
    writeFileSync(join(cwd, path), "Changed\n");
    commit(cwd);
    const result = runChanges(cwd, base);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(readFileSync(join(cwd, "outputs"), "utf8"), `app=${app}\nsite=${site}\n`);
  });
}

test("renaming application input into docs still selects application checks", (t) => {
  const { cwd } = fixture(t);
  writeFileSync(join(cwd, "app.go"), "package app\n");
  commit(cwd);
  const base = git(cwd, "rev-parse", "HEAD");
  git(cwd, "rm", "README.md");
  git(cwd, "mv", "app.go", "README.md");
  commit(cwd);
  const result = runChanges(cwd, base);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(readFileSync(join(cwd, "outputs"), "utf8"), "app=true\nsite=false\n");
});

test("site and prose together do not trigger packaging", (t) => {
  const { cwd, base } = fixture(t);
  mkdirSync(join(cwd, "site"));
  writeFileSync(join(cwd, "site", "index.mdx"), "Docs\n");
  writeFileSync(join(cwd, "README.md"), "Updated\n");
  commit(cwd);
  const result = runChanges(cwd, base);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(readFileSync(join(cwd, "outputs"), "utf8"), "app=false\nsite=true\n");
});

for (const [label, base, extra] of [
  ["manual run", "", {}],
  ["initial push", "0".repeat(40), {}],
  ["release", "", { CI_FORCE_FULL: "true" }],
]) {
  test(`${label} always selects full validation`, (t) => {
    const { cwd } = fixture(t);
    const result = runChanges(cwd, base, extra);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(readFileSync(join(cwd, "outputs"), "utf8"), "app=true\nsite=true\n");
  });
}

test("an unavailable baseline fails rather than silently skipping work", (t) => {
  const { cwd } = fixture(t);
  assert.notEqual(runChanges(cwd, "f".repeat(40)).status, 0);
});

function needs(app, site) {
  return {
    changes: { result: "success", outputs: { app: String(app), site: String(site) } },
    ...Object.fromEntries(["go", "storage", "frontend", "windows"].map((job) => [job, { result: app ? "success" : "skipped" }])),
    site: { result: site ? "success" : "skipped" },
  };
}

function gate(input) {
  return spawnSync(process.execPath, [script, "gate"], {
    encoding: "utf8", env: { ...process.env, CI_NEEDS: JSON.stringify(input) },
  });
}

for (const [app, site] of [[true, true], [true, false], [false, true], [false, false]]) {
  test(`gate accepts successful selected jobs and intentional skips (app=${app}, site=${site})`, () => {
    const result = gate(needs(app, site));
    assert.equal(result.status, 0, result.stderr);
  });
}

for (const job of ["changes", "go", "storage", "frontend", "windows", "site"]) {
  for (const result of ["failure", "cancelled", "skipped"]) {
    test(`gate rejects ${result} for required ${job}`, () => {
      const input = needs(true, true);
      input[job].result = result;
      assert.notEqual(gate(input).status, 0);
    });
  }
}

test("gate rejects missing selection outputs", () => {
  const input = needs(false, false);
  delete input.changes.outputs.app;
  assert.notEqual(gate(input).status, 0);
});

test("gate rejects an unexpected job failure even if the job was not selected", () => {
  const input = needs(false, false);
  input.windows.result = "failure";
  assert.notEqual(gate(input).status, 0);
});
