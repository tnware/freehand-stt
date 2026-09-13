// Website presentation only; this does not exercise native recording/insertion.
import { chromium } from '../../frontend/node_modules/playwright/index.mjs';
import { spawn } from 'node:child_process';
import assert from 'node:assert/strict';

const origin = 'http://127.0.0.1:4398';
const base = process.env.SITE_TEST_BASE || '/';
const server = spawn(process.execPath, ['node_modules/astro/bin/astro.mjs', 'preview', '--ignore-lock', '--host', '127.0.0.1', '--port', '4398'], { stdio: 'ignore' });
let browser;
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(origin + base)).ok) break; } catch {}
    if (i === 99) throw new Error('Preview did not start');
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  await context.route('https://api.github.com/**', route => route.abort());
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(origin + base);
  const demo = page.locator('#preview-dictation freehand-dictation-demo');
  await page.waitForFunction(() => customElements.get('freehand-dictation-demo'));
  assert.equal(await page.locator('[data-preview="dictation"]').getAttribute('aria-pressed'), 'true');
  await demo.locator('[data-demo-replay]').click();
  // Observe real frames: three completed dictations accumulate, then the loop resets.
  const samples = await demo.evaluate(element => new Promise(resolve => {
    const samples = [];
    const started = performance.now();
    const timer = setInterval(() => {
      samples.push({ ms: performance.now() - started, phase: element.dataset.phase, text: element.querySelector('[data-demo-text]').textContent });
      if (performance.now() - started >= 15000) { clearInterval(timer); resolve(samples); }
    }, 50);
  }));
  const transcript = await demo.locator('[data-demo-text]').getAttribute('data-transcript');
  const sentences = transcript.split('\n\n');
  const firstProcessing = samples.find(sample => sample.phase === 'processing');
  assert.ok(firstProcessing.ms < 2200, 'short recording beat');
  assert.equal(firstProcessing.text, '', 'no insertion during first processing');
  for (let i = 1; i <= sentences.length; i++) {
    assert.ok(samples.some(sample => sample.text === sentences.slice(0, i).join('\n\n')), `accumulated sentence ${i}`);
  }
  const full = samples.findIndex(sample => sample.text === transcript);
  assert.ok(full >= 0 && samples.slice(full + 1).some(sample => sample.phase === 'recording' && sample.text === ''), 'automatic loop');
  await demo.locator('[data-demo-toggle]').click();
  const frozen = await demo.locator('[data-demo-text]').textContent();
  await page.waitForTimeout(400);
  assert.equal(await demo.locator('[data-demo-text]').textContent(), frozen, 'pause');
  assert.equal(await demo.getAttribute('data-paused'), 'true');
  for (const tab of ['transcription', 'speech']) {
    await page.locator(`[data-preview="${tab}"]`).click();
    assert.ok(await page.locator(`#preview-${tab}`).isVisible());
    assert.ok(!(await demo.isVisible()));
    await page.locator('.preview-enlarge').click();
    assert.ok(await page.locator('dialog img').isVisible());
    assert.ok(!(await page.locator('[data-dialog-demo]').isVisible()));
    await page.keyboard.press('Escape');
  }
  await page.locator('[data-preview="dictation"]').click();
  await page.locator('.preview-enlarge').click();
  assert.ok(await page.locator('[data-dialog-demo]').isVisible());
  assert.ok(!(await page.locator('dialog img').isVisible()));
  await page.keyboard.press('Escape');
  for (const width of [1440, 768, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `no overflow at ${width}`);
  }
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.reload();
  await page.waitForFunction(() => customElements.get('freehand-dictation-demo'));
  assert.equal(await demo.locator('[data-demo-text]').textContent(), transcript);
  assert.equal(await demo.getAttribute('data-paused'), 'true', 'reduced motion starts static');
  assert.deepEqual(errors, [], 'no uncaught browser errors');
  const noJS = await browser.newContext({ javaScriptEnabled: false });
  const staticPage = await noJS.newPage();
  await staticPage.goto(origin + base);
  assert.equal(await staticPage.locator('#preview-dictation [data-demo-text]').textContent(), transcript);
  assert.ok(!(await staticPage.locator('.preview-controls').isVisible()));
  console.log('PASS short recording, three cumulative insertions, automatic loop, pause, tabs, enlargement/Escape, four viewport widths, reduced motion, no-JS fallback, browser errors');
} finally {
  await browser?.close();
  server.kill();
}
