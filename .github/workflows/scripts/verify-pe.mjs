import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

export function verifyPE(bytes, arch) {
  const machine = { amd64: 0x8664, arm64: 0xaa64 }[arch];
  if (!machine) throw new Error("Unsupported Windows architecture");
  if (bytes.length < 64 || bytes.readUInt16LE(0) !== 0x5a4d) throw new Error("Missing DOS header");
  const offset = bytes.readUInt32LE(0x3c);
  if (offset < 64 || offset + 24 > bytes.length || bytes.readUInt32LE(offset) !== 0x4550) throw new Error("Missing PE header");
  if (bytes.readUInt16LE(offset + 4) !== machine) throw new Error("Executable architecture does not match release target");
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [path, arch] = process.argv.slice(2);
  verifyPE(readFileSync(path), arch);
}
