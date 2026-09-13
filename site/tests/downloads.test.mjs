import test from 'node:test';
import assert from 'node:assert/strict';
import * as downloads from '../src/lib/downloads.mjs';

const release = (keys, tag = 'v-test') => ({ draft: false, tag_name: tag, html_url: `${downloads.RELEASES_URL}/tag/${tag}`, assets: keys.map(key => ({ name: downloads.ASSETS[key].name, browser_download_url: `${downloads.RELEASES_URL}/download/${tag}/${downloads.ASSETS[key].name}` })) });
test('release mapping never mixes versions or invents missing assets', () => {
  const result = downloads.resolveRelease([release(['installer']), release(['macArm'], 'older')]);
  assert.ok(result.assets.installer);
  assert.equal(result.assets.macArm, undefined);
  assert.equal(result.url, `${downloads.RELEASES_URL}/tag/v-test`);
  assert.equal(downloads.resolveRelease([release(Object.keys(downloads.ASSETS))]).assets.macIntel, `${downloads.RELEASES_URL}/download/v-test/freehand-darwin-amd64.zip`);
});
test('malformed and unsafe release data stays unavailable', () => {
  for (const data of [null, {}, [], [null], [{ draft: false }]]) assert.equal(downloads.resolveRelease(data), null);
  const bad = release(['macArm']);
  for (const url of ['javascript:alert(1)', 'https://evil.test/file', `${downloads.RELEASES_URL}/download/older/freehand-darwin-arm64.zip`]) {
    bad.assets[0].browser_download_url = url;
    assert.equal(downloads.resolveRelease([bad]).assets.macArm, undefined);
  }
  bad.html_url = 'javascript:alert(1)';
  assert.equal(downloads.resolveRelease([bad]), null);
  assert.deepEqual(downloads.resolveRelease([release([])]).assets, {});
  assert.equal(downloads.resolveRelease([{...release([]), assets: null}]), null);
});

test('recommendation distinguishes desktop, mobile and genuinely known Mac architecture', () => {
  assert.equal(downloads.recommend({ userAgent: 'Windows NT 10.0; Win64; x64' }).primary, 'installer');
  assert.deepEqual(downloads.recommend({ userAgent: 'Macintosh; Intel Mac OS X 10_15_7' }).choices, ['macArm', 'macIntel']);
  assert.equal(downloads.recommend({ platform: 'macOS', architecture: 'arm', bitness: '64' }).primary, 'macArm');
  assert.equal(downloads.recommend({ platform: 'macOS', architecture: 'x86', bitness: '64' }).primary, 'macIntel');
  for (const userAgent of ['Linux x86_64', 'Android Linux', 'iPhone Mac OS X', 'unknown']) {
    assert.equal(downloads.recommend({ userAgent }).primary, null);
  }
  assert.equal(downloads.recommend({ userAgent: 'Macintosh', maxTouchPoints: 5 }).primary, null);
});
