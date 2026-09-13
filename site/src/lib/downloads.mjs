export const RELEASES_URL = 'https://github.com/tnware/freehand-stt/releases';
export const RELEASE_API = 'https://api.github.com/repos/tnware/freehand-stt/releases?per_page=10';
export const ASSETS = {
  installer: { name: 'freehand-windows-amd64-installer.exe', label: 'Windows installer', detail: 'Windows 11 · x64 · Per-user install' },
  portable: { name: 'freehand-windows-amd64.exe', label: 'Windows portable', detail: 'Windows 11 · x64 · No installation' },
  macArm: { name: 'freehand-darwin-arm64.zip', label: 'Mac · Apple Silicon', detail: 'macOS 13+ · ARM64 · ZIP' },
  macIntel: { name: 'freehand-darwin-amd64.zip', label: 'Mac · Intel', detail: 'macOS 13+ · x64 · ZIP' },
};

// Select exactly one public release, including alpha prereleases, in GitHub's order.
// Never fall back to assets from an older release when this one is partial.
export function resolveRelease(data) {
  if (!Array.isArray(data)) return null;
  const release = data.find(item => item && item.draft === false);
  if (!release || typeof release.tag_name !== 'string' || !release.tag_name || !Array.isArray(release.assets)) return null;
  const tag = encodeURIComponent(release.tag_name);
  const url = `${RELEASES_URL}/tag/${tag}`;
  if (release.html_url !== url) return null;
  const assets = {};
  for (const [key, asset] of Object.entries(ASSETS)) {
    const expected = `${RELEASES_URL}/download/${tag}/${asset.name}`;
    const match = release.assets.find(item => item?.name === asset.name && item.browser_download_url === expected);
    // This expected URL is only a validator, never a fabricated download fallback.
    if (match) assets[key] = match.browser_download_url;
  }
  return { version: release.tag_name, url, assets };
}

export function recommend({ userAgent = '', platform = '', architecture = '', bitness = '', mobile = false, maxTouchPoints = 0 } = {}) {
  const unsupported = { primary: null, choices: [], message: 'Choose a desktop build below. Freehand supports Windows 11 x64 and macOS 13+; Linux and mobile are not supported.' };
  if (mobile || /Android|iPhone|iPad|iPod|Mobile/i.test(userAgent) || (/Mac/i.test(userAgent + platform) && maxTouchPoints > 1)) return unsupported;
  if (/Windows/i.test(platform + userAgent)) {
    if (/arm/i.test(architecture + userAgent) || bitness === '32') return unsupported;
    return { primary: 'installer', choices: ['installer', 'portable'], message: 'For Windows 11 x64: use the installer, or run the portable executable.' };
  }
  if (/macOS|Macintosh|MacIntel|Mac OS X/i.test(platform + userAgent)) {
    // Intel in a Mac UA is also emitted on Apple Silicon. Only explicit client hints identify architecture.
    const primary = /^(arm|arm64|aarch64)$/i.test(architecture) ? 'macArm' : /^(x86|x86_64|amd64)$/i.test(architecture) && bitness === '64' ? 'macIntel' : null;
    return { primary, choices: primary ? [primary] : ['macArm', 'macIntel'], message: primary ? `For your Mac: ${ASSETS[primary].label}.` : 'Choose Apple Silicon or Intel. Check Apple menu → About This Mac for Chip or Processor; this browser cannot reliably identify your Mac architecture.' };
  }
  return unsupported;
}
