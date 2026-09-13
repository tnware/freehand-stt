// Capture real production components with illustrative, service-boundary data.
// Requires npm ci in frontend and site, Playwright Chromium, and generated
// frontend bindings (from the repository root: wails3 generate bindings -ts -i).
import { chromium } from '../../frontend/node_modules/playwright/index.mjs';
import { spawn } from 'node:child_process';
import { copyFileSync, mkdtempSync, rmSync } from 'node:fs';
import { join, basename } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const frontend = fileURLToPath(new URL('../../frontend/', import.meta.url));
const origin = 'http://127.0.0.1:9363';
// Serve the site-owned fixture from a temporary frontend entry so imports share
// the production Vite/Svelte graph. No fixture files remain in the app tree.
const fixture = mkdtempSync(join(frontend, 'tests/browser/.homepage-'));
const fixtureURL = `${origin}/tests/browser/${basename(fixture)}/homepage.html`;
const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--config', 'tests/browser/vite.config.ts', '--port', '9363'], { cwd: frontend, stdio: ['ignore', 'pipe', 'pipe'] });
let serverLog = '';
server.stdout.on('data', chunk => { serverLog += chunk; });
server.stderr.on('data', chunk => { serverLog += chunk; });
let browser;
try {
  for (const name of ["homepage.html", "homepage.ts"]) {
    copyFileSync(new URL(`./app-preview/${name}`, import.meta.url), join(fixture, name));
  }
  for (let attempt = 0; attempt < 100; attempt++) {
    if (server.exitCode !== null) throw new Error(serverLog);
    try { if ((await fetch(fixtureURL)).ok) break; } catch {}
    if (attempt === 99) throw new Error('Screenshot server did not start: ' + serverLog);
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1100, height: 700 },
    deviceScaleFactor: 2,
    colorScheme: 'dark',
    reducedMotion: 'reduce',
  });
  await context.route('**/*', route => new URL(route.request().url()).origin === origin ? route.continue() : route.abort());
  await context.route('**/bindings/**/internal/windowing/service.*', route => route.fulfill({
    contentType: 'application/javascript',
    body: `
      export const SettingsVisible = async () => false;
      export const AboutVisible = async () => false;
      export const ShellReady = async () => {};
      export const OpenSettings = async () => {};
      export const OpenTaskSettings = async () => {};
      export const OpenTaskConnection = async () => {};
      export const OpenConnectionManager = async () => {};
      export const OpenAbout = async () => {};
    `,
  }));
  await context.route('**/bindings/**/internal/buildinfo/service.*', route => route.fulfill({
    contentType: 'application/javascript',
    body: 'export const Current = async () => ({version: ""});',
  }));
  for (const task of ['transcription', 'speech']) {
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => { errors.push(error.message); console.error(error.message); });
    await page.goto(`${fixtureURL}?task=${task}`);
    await page.getByRole('tab', { name: task === 'speech' ? 'Text to speech' : 'Audio file', exact: true }).click();
    if (task === 'speech') {
      await page.getByRole('textbox').filter({ visible: true }).waitFor();
      assert.match(await page.getByRole('textbox').filter({ visible: true }).inputValue(), /^Take the scenic route\./);
      await page.getByRole('button', { name: /Resume/ }).waitFor();
      await page.getByRole('button', { name: 'Text to speech connection status: Not checked', exact: true }).click();
      await page.getByRole('button', { name: 'Check connection', exact: true }).click();
      await page.getByRole('button', { name: 'Text to speech connection status: Reachable', exact: true }).waitFor();
      await page.keyboard.press('Escape');
    } else {
      await page.getByText('Coastal walk.m4a', { exact: true }).waitFor();
      await page.getByText(/I took the coastal path just after sunrise/).waitFor();
    }
    await page.evaluate(() => document.fonts.ready);
    // A neutral pointer click removes keyboard focus left by closing a popover.
    await page.locator('header h1').click();
    await page.mouse.move(1099, 699);
    assert.equal(await page.getByRole('dialog').count(), 0, 'no open setup or connection panels');
    await page.getByRole('button', { name: 'Open About', exact: true }).waitFor();
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth && document.documentElement.scrollHeight <= innerHeight), 'window fits');
    assert.doesNotMatch(await page.locator('body').innerText(), /Example connection|Synthetic|sample text for reviewing|composer keeps its place/);
    assert.deepEqual(errors, [], 'no uncaught browser errors');
    const path = fileURLToPath(new URL(`../src/assets/app-${task}-review.png`, import.meta.url));
    await page.screenshot({ path, animations: 'disabled', caret: 'hide' });
    console.log(`Captured ${task}: ${path}`);
    await page.close();
  }
} finally {
  try { await browser?.close(); } finally {
    server.kill();
    rmSync(fixture, { recursive: true, force: true });
  }
}
