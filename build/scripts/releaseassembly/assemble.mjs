import { createHash } from "node:crypto";
import { lstatSync, mkdirSync, mkdtempSync, readFileSync, renameSync, rmSync, writeFileSync, existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { assets, checksumName } from "./assets.mjs";

const [input, destination] = process.argv.slice(2);
if (!input || !destination || process.argv.length !== 4) throw new Error("Usage: assemble.mjs INPUT OUTPUT");
const output = resolve(destination);
if (existsSync(output)) throw new Error("Release output must not already exist");
// Validate every input before creating anything publishable. Do not follow links.
const contents = assets.map(([source, name]) => {
  const path = join(input, source);
  const stat = lstatSync(path);
  if (!stat.isFile() || stat.size === 0) throw new Error(`Missing, empty or non-regular asset: ${source}`);
  const bytes = readFileSync(path);
  if (!bytes.length) throw new Error(`Empty asset: ${source}`);
  return { name, bytes };
});
mkdirSync(dirname(output), { recursive: true });
const staging = mkdtempSync(join(dirname(output), ".release-"));
try {
  const sums = contents.map(({ name, bytes }) => {
    writeFileSync(join(staging, name), bytes, { flag: "wx" });
    return `${createHash("sha256").update(bytes).digest("hex")}  ${name}`;
  });
  writeFileSync(join(staging, checksumName), sums.join("\n") + "\n", { flag: "wx" });
  renameSync(staging, output);
} finally {
  rmSync(staging, { recursive: true, force: true });
}
