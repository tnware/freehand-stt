import assert from "node:assert/strict";
import test from "node:test";
import { verifyPE } from "./verify-pe.mjs";

function executable(machine) {
  const bytes = Buffer.alloc(88);
  bytes.writeUInt16LE(0x5a4d, 0);
  bytes.writeUInt32LE(64, 0x3c);
  bytes.writeUInt32LE(0x4550, 64);
  bytes.writeUInt16LE(machine, 68);
  return bytes;
}

test("rejects mislabeled Windows release executables", () => {
  verifyPE(executable(0x8664), "amd64");
  verifyPE(executable(0xaa64), "arm64");
  assert.throws(() => verifyPE(executable(0x8664), "arm64"));
  assert.throws(() => verifyPE(executable(0xaa64), "amd64"));
  assert.throws(() => verifyPE(executable(0x14c), "arm64"));
  assert.throws(() => verifyPE(executable(0xaa64), "unknown"));
});

test("rejects malformed or truncated PE headers", () => {
  for (const bytes of [Buffer.alloc(0), Buffer.alloc(88), executable(0xaa64).subarray(0, 70)]) {
    assert.throws(() => verifyPE(bytes, "arm64"));
  }
  const bytes = executable(0xaa64);
  bytes.writeUInt32LE(0xffffffff, 0x3c);
  assert.throws(() => verifyPE(bytes, "arm64"));
});
