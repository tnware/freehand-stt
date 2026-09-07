import { execFileSync } from "node:child_process";
import { appendFileSync } from "node:fs";

function changes() {
  const base = process.env.CI_BASE_SHA;
  if (process.env.CI_FORCE_FULL === "true" || !base || /^0+$/.test(base)) {
    appendFileSync(process.env.GITHUB_OUTPUT, "app=true\nsite=true\n");
    return;
  }
  if (!/^[0-9a-f]{40}$/i.test(base)) throw new Error("Expected a full baseline commit SHA");
  // --no-renames includes both the old and new path; NULs preserve odd filenames.
  // Diff failure (including an unavailable baseline) fails the job, never skips it.
  const files = execFileSync("git", ["diff", "--no-renames", "--name-only", "-z", base, "HEAD", "--"], { encoding: "utf8" }).split("\0").filter(Boolean);
  // Only known prose/site inputs are exempt; unknown paths fail toward coverage.
  const app = files.some((path) => !path.startsWith("site/") && path !== "README.md" && path !== "branding/README.md");
  const site = files.some((path) =>
    ["site/", ".github/", "internal/compatibility/", "build/scripts/compatibility/", "branding/providers/"].some((prefix) => path.startsWith(prefix)) ||
    ["branding/freehand-mark.svg", "go.mod", "go.sum"].includes(path),
  );
  appendFileSync(process.env.GITHUB_OUTPUT, `app=${app}\nsite=${site}\n`);
}

function gate() {
  const needs = JSON.parse(process.env.CI_NEEDS);
  if (needs.changes?.result !== "success") throw new Error("Workload selection did not succeed");
  const { app, site } = needs.changes.outputs;
  if (![app, site].every((value) => value === "true" || value === "false")) {
    throw new Error("Missing or invalid workload selection");
  }
  for (const job of ["go", "storage", "frontend", "windows", "site"]) {
    const selected = (job === "site" ? site : app) === "true";
    const expected = selected ? "success" : "skipped";
    if (needs[job]?.result !== expected) throw new Error(`${job}: expected ${expected}, got ${needs[job]?.result}`);
  }
  console.log("All selected validation jobs passed.");
}

if (process.argv[2] === "changes") {
  changes();
} else if (process.argv[2] === "gate") {
  gate();
} else {
  throw new Error("Expected changes or gate command");
}
