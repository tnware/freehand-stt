import { readFileSync } from "node:fs";
import { validatePublicAssetNames } from "./assets.mjs";

// Consume `gh release view --json assets` on stdin; fail closed on bad metadata.
const release = JSON.parse(readFileSync(0, "utf8"));
if (!release || !Array.isArray(release.assets)) throw new Error("Malformed release assets");
validatePublicAssetNames(release.assets.map((asset) => asset?.name));
