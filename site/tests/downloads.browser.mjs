// Run against a fresh production build. Synthetic Mac assets are fixtures, not published releases.
import { chromium } from '../../frontend/node_modules/playwright/index.mjs';
import { spawn } from 'node:child_process';
import assert from 'node:assert/strict';
import { ASSETS, RELEASES_URL } from '../src/lib/downloads.mjs';
const port = 4397;
const base = process.env.SITE_TEST_BASE || '/';
const origin = `http://127.0.0.1:${port}`;
const server = spawn(process.execPath, ['node_modules/astro/bin/astro.mjs', 'preview', '--ignore-lock', '--host', '127.0.0.1', '--port', String(port)], { stdio: 'ignore' });
let browser;
let count = 0;
const fixture = keys => [{ draft: false, tag_name: 'v-fixture', html_url: `${RELEASES_URL}/tag/v-fixture`, assets: keys.map(key => ({ name: ASSETS[key].name, browser_download_url: `${RELEASES_URL}/download/v-fixture/${ASSETS[key].name}` })) }];
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(origin + base)).ok) break; } catch {}
    if (i === 99) throw new Error('Preview did not start');
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  browser = await chromium.launch({ headless: true });
  const cases = [
    { name: 'Windows installer and portable', ua: 'Windows NT 10.0; Win64; x64', primary: 'installer' },
    { name: 'Apple Silicon hint', ua: 'Macintosh; Intel Mac OS X', hints: { platform: 'macOS', architecture: 'arm', bitness: '64' }, primary: 'macArm' },
    { name: 'Intel hint', ua: 'Macintosh; Intel Mac OS X', hints: { platform: 'macOS', architecture: 'x86', bitness: '64' }, primary: 'macIntel' },
    { name: 'Ambiguous Intel Mac UA', ua: 'Macintosh; Intel Mac OS X', choices: true },
    { name: 'Linux', ua: 'Linux x86_64' },
    { name: 'Android', ua: 'Linux; Android 14; Mobile' },
    { name: 'iPhone', ua: 'iPhone; CPU iPhone OS like Mac OS X' },
    { name: 'iPad desktop UA', ua: 'Macintosh; Intel Mac OS X', touch: 5 },
    { name: 'Unknown', ua: 'unknown' },
    { name: 'Windows-only release on Mac', ua: 'Macintosh', keys: ['installer', 'portable'], choices: true },
    { name: 'Partial release', ua: 'Windows NT 10.0', keys: ['portable'] },
    { name: 'Empty assets', ua: 'Windows NT 10.0', keys: [] },
    { name: 'Empty releases', data: [] },
    { name: 'Malformed response', data: {} },
    { name: 'Malformed assets', data: [{...fixture([])[0], assets: null}] },
    { name: 'Network failure', failure: true },
    { name: 'HTTP failure', status: 503 },
    { name: 'Timeout', hang: true },
    { name: 'No JavaScript', nojs: true },
  ];
  for (const c of cases) {
    const context = await browser.newContext({ javaScriptEnabled: !c.nojs, userAgent: c.ua || 'unknown' });
    await context.addInitScript(({ hints, touch }) => {
      Object.defineProperty(navigator, 'userAgentData', { value: hints ? { ...hints, getHighEntropyValues: async () => hints } : undefined });
      Object.defineProperty(navigator, 'maxTouchPoints', { value: touch || 0 });
    }, c);
    await context.route('https://api.github.com/**', route => c.hang ? new Promise(() => {}) : c.failure ? route.abort() : route.fulfill({ status: c.status || 200, contentType: 'application/json', body: JSON.stringify(c.data ?? fixture(c.keys ?? Object.keys(ASSETS))) }));
    const page = await context.newPage();
    await page.goto(origin + base + 'download/');
    if (!c.nojs) await page.waitForFunction(() => document.querySelector('[data-release-status]')?.getAttribute('data-settled') === 'true', null, { timeout: 10000 });
    for (const key of Object.keys(ASSETS)) assert.ok(await page.locator(`#all-downloads [data-asset="${key}"]`).isVisible(), `${c.name}: ${key} visible`);
    for (const platform of ['windows', 'mac']) {
      const group = page.locator(`[data-download-platform="${platform}"]`);
      assert.equal(await group.locator('[data-asset]').count(), 2);
      assert.ok(await group.locator(`[data-platform-icon="${platform}"]`).isVisible());
    }
    for (const row of await page.locator('[data-recommendation] .release-row:visible').all()) {
      const key = await row.getAttribute('data-asset');
      assert.ok(await row.locator(`[data-platform-icon="${key.startsWith('mac') ? 'mac' : 'windows'}"]`).isVisible(), `${c.name}: icon survives release update`);
      assert.match(await row.locator('[data-asset-label]').innerText(), /Download|View release|Check GitHub availability/);
    }
    if (c.primary) assert.ok((await page.locator('[data-primary-download]').getAttribute('href')).endsWith(ASSETS[c.primary].name), c.name);
    else assert.ok(!(await page.locator('[data-primary-download]').getAttribute('href')).includes('/download/v-'), c.name);
    if (c.choices) for (const key of ['macArm', 'macIntel']) assert.ok(await page.locator(`[data-recommendation] [data-asset="${key}"]`).isVisible());
    if (c.keys) for (const key of Object.keys(ASSETS).filter(key => !c.keys.includes(key))) assert.match(await page.locator(`#all-downloads [data-asset="${key}"]`).innerText(), /Not in this release/);
    if (c.nojs || c.failure || c.status || c.hang || c.data) assert.equal(await page.locator('#all-downloads [data-asset="macArm"]').getAttribute('href'), RELEASES_URL);
    await context.close();
    console.log(`PASS ${c.name}`); count++;
  }
  const page = await browser.newPage();
  for (const width of [390, 768, 1365]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto(origin + base + 'download/');
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `download layout at ${width}`);
    for (const key of Object.keys(ASSETS)) assert.ok(await page.locator(`#all-downloads [data-asset="${key}"]`).isVisible());
  }
  await page.goto(origin + base + 'backends/');
  for (const row of await page.locator('[data-backend-name]').all()) {
    const capabilities = (await row.getAttribute('data-capabilities')).split(' ');
    const cells = row.locator('.capability-cell');
    const keys = ['microphone', 'realtime', 'cleanup', 'playback'];
    for (let i = 0; i < keys.length; i++) {
      assert.equal(await cells.nth(i).locator('.sr-only').innerText(), capabilities.includes(keys[i]) ? 'Supported' : 'Not supported');
      assert.ok(await cells.nth(i).locator('svg[aria-hidden="true"]').isVisible());
    }
  }
  await page.locator('#compare summary').click();
  assert.ok(await page.locator('#compare .support-indicator').count() > 0);
  await page.locator('select[aria-label="Filter by capability"]').selectOption('realtime');
  for (const row of await page.locator('[data-backend-name]:visible').all()) assert.match(await row.getAttribute('data-capabilities'), /realtime/);
  console.log('PASS platform groups, release icons, responsive layouts, and accessible capability indicators');
  await page.goto(origin + base);
  assert.match(await page.title(), /Windows and macOS/);
  assert.equal(await page.locator('[data-home-download]').getAttribute('href'), base + 'download/');
  console.log(`PASS homepage metadata and base path; ${count} release scenarios plus layout and capability checks passed`);
} finally {
  await browser?.close();
  server.kill();
}
