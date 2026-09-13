import { ASSETS, RELEASES_URL, RELEASE_API, recommend, resolveRelease } from './downloads.mjs';

export async function enhanceDownloads() {
  const nav = navigator;
  let hints = {};
  try {
    hints = await Promise.race([
      nav.userAgentData?.getHighEntropyValues(['architecture', 'bitness', 'platform']) ?? {},
      new Promise(resolve => setTimeout(() => resolve({}), 700)),
    ]);
  } catch { /* Architecture remains explicitly unknown. */ }
  const recommendation = recommend({ userAgent: nav.userAgent, maxTouchPoints: nav.maxTouchPoints, mobile: nav.userAgentData?.mobile, ...hints });
  const home = document.querySelector('[data-home-download]');
  if (home) home.textContent = recommendation.primary ? `Downloads for ${recommendation.primary.startsWith('mac') ? 'macOS' : 'Windows'} →` : 'Choose your download →';
  const region = document.querySelector('[data-recommendation]');
  if (!region) return;
  document.querySelector('[data-platform-message]').textContent = recommendation.message;
  for (const key of recommendation.choices) {
    const link = region.querySelector(`[data-asset="${key}"]`);
    if (link) link.hidden = false;
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 5000);
  let release = null;
  try {
    const response = await fetch(RELEASE_API, { signal: controller.signal, headers: { Accept: 'application/vnd.github+json' } });
    if (response.ok) release = resolveRelease(await response.json());
  } catch { /* Static links remain usable on timeout, rate limits and malformed JSON. */ }
  finally { clearTimeout(timer); }
  const status = document.querySelector('[data-release-status]');
  status.textContent = release ? `Latest public release: ${release.version}. Availability below is for this release only.` : 'Release availability could not be checked. Open GitHub Releases to see the available downloads.';
  document.querySelector('[data-release-version]').textContent = release?.version ?? 'Check release availability';
  for (const link of document.querySelectorAll('[data-asset]')) {
    const key = link.dataset.asset;
    link.href = release?.assets[key] ?? release?.url ?? RELEASES_URL;
    const label = link.querySelector('[data-asset-label]') ?? link;
    label.textContent = `${ASSETS[key].label} — ${release?.assets[key] ? 'Download' : release ? 'Not in this release · View release' : 'Check GitHub availability'}`;
  }
  const primary = document.querySelector('[data-primary-download]');
  const key = recommendation.primary;
  if (key && release?.assets[key]) {
    primary.href = release.assets[key];
    primary.textContent = `Download ${ASSETS[key].label}`;
  } else {
    primary.href = key ? release?.url ?? RELEASES_URL : '#all-downloads';
    primary.textContent = key ? `${ASSETS[key].label}: ${release ? 'not in this release' : 'check availability'}` : 'Choose your desktop build';
  }
  status.dataset.settled = 'true';
}
