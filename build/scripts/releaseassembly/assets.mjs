export const assets = Object.freeze([
  ["freehand.exe", "freehand-windows-amd64.exe"],
  ["freehand-amd64-installer.exe", "freehand-windows-amd64-installer.exe"],
  ["freehand-darwin-arm64.zip", "freehand-darwin-arm64.zip"],
  ["freehand-darwin-amd64.zip", "freehand-darwin-amd64.zip"],
].sort((a, b) => a[1].localeCompare(b[1])).map(Object.freeze));
export const checksumName = "SHA256SUMS";
export const publicAssetNames = Object.freeze([...assets.map(([, name]) => name), checksumName]);

export function validatePublicAssetNames(names) {
  if (!Array.isArray(names)) throw new Error("Malformed public asset names: expected an array");
  const seen = new Set();
  for (const name of names) {
    if (typeof name !== "string" || !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(name)) {
      throw new Error("Malformed public asset name");
    }
    if (seen.has(name)) throw new Error(`Duplicate public asset: ${name}`);
    if (!publicAssetNames.includes(name)) throw new Error(`Unexpected public asset: ${name}`);
    seen.add(name);
  }
  const missing = publicAssetNames.filter((name) => !seen.has(name));
  if (missing.length) throw new Error(`Missing public assets: ${missing.join(", ")}`);
}
