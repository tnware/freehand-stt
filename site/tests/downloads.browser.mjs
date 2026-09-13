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
    if (c.primary) assert.ok((await page.locator('[data-primary-download]').getAttribute('href')).endsWith(ASSETS[c.primary].name), c.name);
    else assert.ok(!(await page.locator('[data-primary-download]').getAttribute('href')).includes('/download/v-'), c.name);
    if (c.choices) for (const key of ['macArm', 'macIntel']) assert.ok(await page.locator(`[data-recommendation] [data-asset="${key}"]`).isVisible());
    if (c.keys) for (const key of Object.keys(ASSETS).filter(key => !c.keys.includes(key))) assert.match(await page.locator(`#all-downloads [data-asset="${key}"]`).innerText(), /Not in this release/);
    if (c.nojs || c.failure || c.status || c.hang || c.data) assert.equal(await page.locator('#all-downloads [data-asset="macArm"]').getAttribute('href'), RELEASES_URL);
    await context.close();
    console.log(`PASS ${c.name}`); count++;
  }
  const page = await browser.newPage();
  await page.goto(origin + base);
  assert.match(await page.title(), /Windows and macOS/);
  assert.equal(await page.locator('[data-home-download]').getAttribute('href'), base + 'download/');
  console.log(`PASS homepage metadata and base path; ${count + 1} browser scenarios passed`);
} finally {
  await browser?.close();
  server.kill();
}
