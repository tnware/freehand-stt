// Experimental distro-specific packages. This command never publishes a release.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export function packageVersion(config) {
  const version = config.match(/^\s+version: "([^"\n]+)"/m)?.[1];
  if (!version || !/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$/.test(version)) {
    throw new Error('Expected a semantic version in build/config.yml');
  }
  // Debian sorts prereleases before the final version.
  return version.replace('-', '~');
}

export function distroName(release) {
  const field = (name) => release.match(new RegExp(`^${name}=["']?([^"'\\n]+)["']?$`, 'm'))?.[1];
  const id = field('ID'), version = field('VERSION_ID');
  if (id === 'debian' && version === '13') return 'debian13';
  if (id === 'ubuntu' && version === '24.04') return 'ubuntu24.04';
  throw new Error('Experimental packaging requires Debian 13 or Ubuntu 24.04');
}

export function verifyELF(bytes) {
  if (bytes.length < 64 || bytes.toString('hex', 0, 4) !== '7f454c46' || bytes[4] !== 2 || bytes[5] !== 1 || bytes.readUInt16LE(18) !== 62 || ![2, 3].includes(bytes.readUInt16LE(16))) {
    throw new Error('Package requires a Linux x86_64 ELF executable');
  }
}

export function verifyDynamicPaths(dynamic) {
  for (const line of dynamic.split('\n')) {
    if (/\((?:RPATH|RUNPATH)\)/.test(line)) {
      const paths = line.match(/\[([^\]]*)\]/)?.[1]?.split(':');
      if (!paths?.length || paths.some(path => !['/usr/lib', '/lib', '/usr/lib/x86_64-linux-gnu', '/lib/x86_64-linux-gnu'].includes(path))) {
        throw new Error('Refusing to package an executable with a non-system runtime library path');
      }
    }
  }
}

export function buildPackage(root = process.cwd()) {
  if (process.platform !== 'linux' || process.arch !== 'x64') throw new Error('Package on native Linux x86_64');
  root = resolve(root);
  const distro = distroName(readFileSync('/etc/os-release', 'utf8'));
  const version = packageVersion(readFileSync(join(root, 'build/config.yml'), 'utf8'));
  const binary = join(root, 'bin/freehand');
  verifyELF(readFileSync(binary));
  verifyDynamicPaths(execFileSync('readelf', ['-d', binary], { encoding: 'utf8' }));
  const work = mkdtempSync(join(tmpdir(), 'freehand-deb-'));
  const run = (command, args, options = {}) => execFileSync(command, args, { encoding: 'utf8', ...options });
  try {
    const stage = join(work, 'package');
    const install = (source, target, mode = 0o644) => {
      const dest = join(stage, target);
      mkdirSync(dirname(dest), { recursive: true });
      copyFileSync(join(root, source), dest);
      chmodSync(dest, mode);
    };
    install('bin/freehand', 'usr/bin/freehand', 0o755);
    install('build/appicon.png', 'usr/share/icons/hicolor/1024x1024/apps/freehand.png');
    install('build/linux/freehand.desktop', 'usr/share/applications/freehand.desktop');
    install('LICENSE', 'usr/share/doc/freehand/copyright');
    run('desktop-file-validate', [join(stage, 'usr/share/applications/freehand.desktop')]);
    mkdirSync(join(work, 'debian'));
    writeFileSync(join(work, 'debian/control'), 'Source: freehand\nMaintainer: tnware <noreply@github.com>\n\nPackage: freehand\nArchitecture: amd64\nDescription: Freehand speech client\n');
    // Resolve minimum versions from THIS distro's linked ELF, never infer
    // Ubuntu compatibility from a newer Debian glibc build (or vice versa).
    const shlibs = run('dpkg-shlibdeps', ['-O', '-e' + binary], { cwd: work }).trim();
    const linked = shlibs.match(/^shlibs:Depends=(.+)$/m)?.[1];
    if (!linked) throw new Error('Native dependency resolution returned no dependencies');
    const dependencies = [...new Set([...linked.split(', '), 'libgtk-4-1', 'libwebkitgtk-6.0-4', 'libasound2t64', 'libpulse0', 'curl', 'ca-certificates', 'libgomp1', 'libstdc++6', 'libssl3t64'])].join(', ');
    mkdirSync(join(stage, 'DEBIAN'));
    writeFileSync(join(stage, 'DEBIAN/control'), `Package: freehand\nVersion: ${version}-1\nArchitecture: amd64\nSection: sound\nPriority: optional\nMaintainer: tnware <noreply@github.com>\nHomepage: https://github.com/tnware/freehand-stt\nDepends: ${dependencies}\nDescription: Experimental Freehand speech client\n Transcribe and generate speech using explicitly configured services.\n Linux qualification build with explicit Copy delivery.\n`);
    const artifact = join(root, 'bin', `freehand_${version}-1_${distro}_amd64.deb`);
    run('dpkg-deb', ['--root-owner-group', '--build', stage, artifact]);
    if (run('dpkg-deb', ['--field', artifact, 'Version']).trim() !== `${version}-1`) throw new Error('Package version mismatch');
    if (run('dpkg-deb', ['--field', artifact, 'Architecture']).trim() !== 'amd64') throw new Error('Package architecture mismatch');
    const digest = createHash('sha256').update(readFileSync(artifact)).digest('hex');
    writeFileSync(artifact + '.sha256', `${digest}  ${artifact.split('/').at(-1)}\n`);
    return artifact;
  } finally {
    rmSync(work, { recursive: true, force: true });
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) console.log(buildPackage());
