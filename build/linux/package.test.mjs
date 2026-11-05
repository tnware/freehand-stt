import test from 'node:test';
import assert from 'node:assert/strict';
import { packageVersion, distroName, verifyELF, verifyDynamicPaths, buildPackage } from './package.mjs';

test('version comes from the application configuration; prereleases sort before stable', () => {
  assert.equal(packageVersion('version: \'3\'\ninfo:\n  version: "1.2.3" # source\n'), '1.2.3');
  assert.equal(packageVersion('  version: "1.2.3-alpha.4"'), '1.2.3~alpha.4');
  for (const value of ['', '  version: "latest"', '  version: "01.2.3"', '  version: "1.2.3\nDepends: evil"']) assert.throws(() => packageVersion(value));
});
test('packages only the explicitly targeted distro versions', () => {
  assert.equal(distroName('ID=debian\nVERSION_ID="13"\n'), 'debian13');
  assert.equal(distroName('ID=ubuntu\nVERSION_ID="24.04"\n'), 'ubuntu24.04');
  for (const value of ['ID=debian\nVERSION_ID="12"', 'ID=ubuntu\nVERSION_ID="22.04"', 'ID=fedora\nVERSION_ID=43', 'ID=ubuntu\nVERSION_ID=26.04']) assert.throws(() => distroName(value));
});
test('fails closed for truncated, non-ELF, wrong-endian and wrong-architecture binaries', () => {
  const elf = Buffer.alloc(64);
  elf.write('7f454c46', 0, 'hex'); elf[4] = 2; elf[5] = 1;
  elf.writeUInt16LE(3, 16); elf.writeUInt16LE(62, 18);
  assert.doesNotThrow(() => verifyELF(elf));
  for (const [offset, value] of [[0, 0], [4, 1], [5, 2], [16, 1], [18, 183]]) {
    const bad = Buffer.from(elf); bad[offset] = value;
    assert.throws(() => verifyELF(bad));
  }
  assert.throws(() => verifyELF(elf.subarray(0, 20)));
});


test('rejects rootless or relative runtime library paths before packaging', () => {
  assert.doesNotThrow(() => verifyDynamicPaths(''));
  assert.doesNotThrow(() => verifyDynamicPaths('0 (RUNPATH) Library runpath: [/usr/lib/x86_64-linux-gnu]'));
  for (const path of ['/workspace/shared/deps/lib', '$ORIGIN', '', '/usr/lib:/tmp', '.']) {
    assert.throws(() => verifyDynamicPaths(`0 (RUNPATH) Library runpath: [${path}]`));
    assert.throws(() => verifyDynamicPaths(`0 (RPATH) Library rpath: [${path}]`));
  }
});

// This validates the archive machinery, not Freehand desktop acceptance.
import { execFileSync, spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';

const nativeTools = process.platform === 'linux' && process.arch === 'x64' &&
  ['cc', 'dpkg-shlibdeps', 'dpkg-deb', 'desktop-file-validate', 'readelf'].every(tool => spawnSync('sh', ['-c', `command -v ${tool}`]).status === 0);
let targetedDistro = false;
try { distroName(readFileSync('/etc/os-release', 'utf8')); targetedDistro = true; } catch {}
test('native fixture package has derived metadata, root-owned payload and matching checksum', { skip: !nativeTools || !targetedDistro || spawnSync('dpkg-query', ['--show', 'libc6']).status !== 0 ? 'Requires native packaging tools and an installed Debian package database' : false }, (t) => {
  const root = mkdtempSync(join(tmpdir(), 'freehand-package-test-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  for (const dir of ['bin', 'build/linux']) mkdirSync(join(root, dir), { recursive: true });
  writeFileSync(join(root, 'build/config.yml'), 'info:\n  version: "1.2.3-alpha.4"\n');
  writeFileSync(join(root, 'LICENSE'), 'Fixture license\n');
  writeFileSync(join(root, 'build/appicon.png'), 'Fixture icon');
  copyFileSync(fileURLToPath(new URL('./freehand.desktop', import.meta.url)), join(root, 'build/linux/freehand.desktop'));
  execFileSync('cc', ['-x', 'c', '-', '-o', join(root, 'bin/freehand')], { input: '#include <stdio.h>\nint main(void) { puts("fixture"); return 0; }\n' });
  const previousUmask = process.umask(0o077);
  let artifact;
  try { artifact = buildPackage(root); } finally { process.umask(previousUmask); }
  assert.match(artifact, /freehand_1\.2\.3~alpha\.4-1_(debian13|ubuntu24\.04)_amd64\.deb$/);
  const control = execFileSync('dpkg-deb', ['--field', artifact], { encoding: 'utf8' });
  for (const field of ['Version: 1.2.3~alpha.4-1', 'Architecture: amd64', 'libgtk-4-1', 'libwebkitgtk-6.0-4', 'libasound2t64', 'libpulse0', 'curl', 'ca-certificates', 'libgomp1', 'libstdc++6', 'libssl3t64']) assert.ok(control.includes(field), field);
  assert.match(control, /libc6 \(>= /);
  const listing = execFileSync('dpkg-deb', ['--contents', artifact], { encoding: 'utf8' });
  assert.match(listing, /-rwxr-xr-x root\/root.*\.\/usr\/bin\/freehand/);
  for (const line of listing.split('\n').filter(line => line.startsWith('d'))) {
    assert.match(line, /^drwxr-xr-x root\/root/);
  }
  const extract = join(root, 'extracted');
  execFileSync('dpkg-deb', ['--extract', artifact, extract]);
  assert.deepEqual(readFileSync(join(extract, 'usr/bin/freehand')), readFileSync(join(root, 'bin/freehand')));
  assert.equal(readFileSync(artifact + '.sha256', 'utf8').split(' ')[0], createHash('sha256').update(readFileSync(artifact)).digest('hex'));
});
