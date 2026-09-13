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
  assert.deepEqual(await page.locator('[data-preview]').evaluateAll(elements => elements.map(e => e.dataset.preview)), ['dictation', 'realtime', 'transcription', 'speech']);
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
  await page.locator('[data-preview="realtime"]').click();
  const realtime = page.locator('#preview-realtime freehand-dictation-demo');
  assert.equal(await realtime.locator('.realtime-note a').getAttribute('href'), base + 'docs/guides/live-transcription/');
  await realtime.locator('[data-demo-replay]').click();
  const liveSamples = await realtime.evaluate(element => new Promise(resolve => {
    const samples = [];
    const started = performance.now();
    const timer = setInterval(() => {
      samples.push({ ms: performance.now() - started, phase: element.dataset.phase, text: element.querySelector('[data-demo-text]').textContent, caption: element.querySelector('[data-live-caption]').textContent, scroll: element.querySelector('.overlay-live-caption').scrollLeft });
      if (performance.now() - started >= 15000) { clearInterval(timer); resolve(samples); }
    }, 50);
  }));
  assert.ok(liveSamples.some(s => s.phase === 'recording' && s.text === '' && s.caption.startsWith('Start')), 'live preview before insertion');
  assert.ok(liveSamples.some(s => s.phase === 'recording' && s.scroll > 0), 'long captions reveal newest words');
  const liveProcessing = liveSamples.find(s => s.phase === 'processing');
  const liveDone = liveSamples.find(s => s.phase === 'done');
  const completedDone = samples.find(s => s.phase === 'done');
  assert.equal(liveProcessing.text, '', 'live partials never enter the document');
  assert.ok(liveDone.ms - liveProcessing.ms < completedDone.ms - firstProcessing.ms, 'shorter illustrative finalization/insertion beat');
  const liveTranscript = await realtime.locator('[data-demo-text]').getAttribute('data-transcript');
  const liveFull = liveSamples.findIndex(s => s.text === liveTranscript);
  assert.ok(liveFull >= 0 && liveSamples.slice(liveFull + 1).some(s => s.phase === 'recording' && s.text === ''), 'realtime loop');
  await realtime.locator('[data-demo-toggle]').click();
  await page.locator('.preview-enlarge').click();
  assert.ok(await page.locator('[data-dialog-realtime]').isVisible());
  assert.ok(!(await page.locator('[data-dialog-demo]').isVisible()));
  assert.ok(!(await page.locator('dialog img').isVisible()));
  await page.keyboard.press('Escape');
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
  for (const width of [1440, 1024, 801]) {
    await page.setViewportSize({ width, height: 1000 });
    const rows = await page.locator('.capability-panel').evaluateAll(panels => panels.map(panel =>
      ['h3', ':scope > p', '.capability-detail', '.provider-links', ':scope > .text-link'].map(selector => panel.querySelector(selector).getBoundingClientRect().top)));
    for (let row = 0; row < rows[0].length; row++) {
      const positions = rows.map(panel => panel[row]);
      assert.ok(Math.max(...positions) - Math.min(...positions) < 1, `capability row ${row} aligned at ${width}`);
    }
  }
  for (const width of [1440, 768, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `no overflow at ${width}`);
    await page.locator('[data-preview="realtime"]').click();
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `no realtime overflow at ${width}`);
    await page.locator('[data-preview="dictation"]').click();
  }
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.reload();
  await page.waitForFunction(() => customElements.get('freehand-dictation-demo'));
  assert.equal(await demo.locator('[data-demo-text]').textContent(), transcript);
  assert.equal(await demo.getAttribute('data-paused'), 'true', 'reduced motion starts static');
  await page.locator('[data-preview="realtime"]').click();
  assert.equal(await realtime.locator('[data-demo-text]').textContent(), liveTranscript);
  assert.equal(await realtime.getAttribute('data-paused'), 'true', 'realtime reduced motion starts static');
  await page.goto(origin + base + 'features/');
  await page.waitForFunction(() => customElements.get('freehand-overlay-explorer'));
  assert.equal(await page.locator('.desktop-navigation a[aria-current="page"]').textContent(), 'Features');
  const layouts = await page.locator('[data-layout]').evaluateAll(elements => elements.map(e => e.dataset.layout));
  const positions = await page.locator('[data-position-control] option').evaluateAll(elements => elements.map(e => e.value));
  assert.equal(layouts.length, 4);
  assert.equal(positions.length, 6);
  assert.equal(await page.locator('.overlay-setting-icon svg[aria-hidden="true"]').count(), 3, 'hero settings retain decorative icons');
  assert.equal(await page.locator('.feature-list article').count(), 6, 'supporting features remain text-first');
  assert.equal(await page.locator('.feature-list svg, .page-heading svg').count(), 0, 'no competing illustrations or one-off title icon');
  for (const width of [1440, 800, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    for (const icon of await page.locator('.overlay-setting-icon').all()) assert.ok(await icon.isVisible(), `hero icon visible at ${width}`);
    for (const layout of layouts) {
      await page.locator(`[data-layout="${layout}"]`).click();
      assert.equal(await page.locator('[data-layout-panel]:not([hidden])').count(), 1);
      assert.equal(await page.locator('[data-layout-panel]:not([hidden])').getAttribute('data-layout-panel'), layout);
      for (const position of positions) {
        await page.locator('[data-position-control]').selectOption(position);
        const bounds = await page.locator('freehand-overlay-explorer').evaluate(element => {
          const stage = element.querySelector('.overlay-stage').getBoundingClientRect();
          const overlay = element.querySelector('[data-layout-panel]:not([hidden])').getBoundingClientRect();
          return { inside: overlay.left >= stage.left && overlay.right <= stage.right && overlay.top >= stage.top && overlay.bottom <= stage.bottom, overflow: document.documentElement.scrollWidth > innerWidth };
        });
        assert.ok(bounds.inside && !bounds.overflow, `${width}: ${layout} at ${position} fits`);
      }
    }
  }
  const featureLinks = await page.locator('main a').evaluateAll(elements => elements.map(e => e.href));
  for (const link of featureLinks) {
    const response = await page.request.get(link);
    assert.ok(response.ok(), `feature guide reachable: ${link}`);
    const hash = new URL(link).hash;
    if (hash) assert.ok(await page.evaluate(({ html, hash }) => Boolean(new DOMParser().parseFromString(html, 'text/html').getElementById(decodeURIComponent(hash.slice(1)))), { html: await response.text(), hash }), `guide section exists: ${link}`);
  }
  assert.deepEqual(errors, [], 'no uncaught browser errors');
  const noJS = await browser.newContext({ javaScriptEnabled: false });
  const staticPage = await noJS.newPage();
  await staticPage.goto(origin + base);
  assert.equal(await staticPage.locator('#preview-dictation [data-demo-text]').textContent(), transcript);
  assert.ok(!(await staticPage.locator('.preview-controls').isVisible()));
  await staticPage.goto(origin + base + 'features/');
  assert.ok(await staticPage.locator('[data-layout-panel="capsule"]').isVisible());
  assert.ok(!(await staticPage.locator('.explorer-controls').isVisible()));
  assert.equal(await staticPage.locator('.overlay-setting-icon svg').count(), 3, 'hero icons work without JavaScript');
  console.log('PASS app previews, capability alignment, responsive layouts, Features overlay layouts/positions and guide links, reduced motion, no-JS fallback, browser errors');
} finally {
  await browser?.close();
  server.kill();
}
