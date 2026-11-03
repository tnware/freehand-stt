import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { qualifyShell, smokeEnvironment, readyMarker } from './smoke.mjs';
const limits = { timeoutMs: 1500, settleMs: 30, cleanupMs: 20 };
const probe = (code, options = {}) => qualifyShell(process.execPath, ['-e', code], { ...limits, ...options });

test('accepts exact frontend marker from a live owned process', async () => {
  const result = await probe(`console.log('${readyMarker}'); setInterval(()=>{},1000)`);
  assert.equal(result.frontendBridgeReady, true);
});
test('requires a complete exact line, not a substring', async () => {
  await assert.rejects(probe(`console.log('prefix ${readyMarker}');`), /exited/);
});
test('rejects an executable that exits without frontend readiness', async () => {
  await assert.rejects(probe('process.exit(0)'), /exited/);
});
test('rejects readiness immediately followed by a crash', async () => {
  await assert.rejects(probe(`console.log('${readyMarker}'); process.exit(1)`), /exited/);
});
test('bounds silent hung launches', async () => {
  await assert.rejects(probe('setInterval(()=>{},1000)', { timeoutMs: 80 }), /timed out/);
});
test('bounds combined process output', async () => {
  await assert.rejects(probe("process.stderr.write('x'.repeat(2000)); setInterval(()=>{},1000)", { maxOutput: 1000 }), /output exceeded/);
});
test('strips inherited credentials and runtime/sandbox overrides', () => {
  const env = smokeEnvironment({ HOME: '/tmp/owned', DISPLAY: ':1', LD_PRELOAD: 'bad', WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS: '1', GITHUB_TOKEN: 'secret', HTTPS_PROXY: 'secret', WAYLAND_DISPLAY: 'wayland-1' });
  assert.deepEqual(env, { HOME: '/tmp/owned', DISPLAY: ':1', FREEHAND_LINUX_SHELL_SMOKE: '1' });
});
test('rejects PATH-selected executables', () => {
  assert.throws(() => qualifyShell('freehand'), /absolute/);
});

test('forced cleanup terminates a SIGTERM-resistant owned process group', async () => {
  const dir = mkdtempSync(join(tmpdir(), 'freehand-smoke-test-'));
  try {
    const pidFile = join(dir, 'child.pid');
    const childCode = "process.on('SIGTERM',()=>{}); console.log('child-ready'); setInterval(()=>{},1000)";
    const parentCode = `
      const {spawn}=require('node:child_process');
      const fs=require('node:fs');
      process.on('SIGTERM',()=>{});
      const child=spawn(process.execPath,['-e',${JSON.stringify(childCode)}],{stdio:['ignore','pipe','ignore']});
      child.stdout.once('data',()=>{ fs.writeFileSync(${JSON.stringify(pidFile)}, String(child.pid)); console.log('${readyMarker}'); });
      setInterval(()=>{},1000);
    `;
    await probe(parentCode, { timeoutMs: 3000 });
    const pid = Number(readFileSync(pidFile, 'utf8'));
    let running = true;
    for (let attempt = 0; attempt < 30 && running; attempt++) {
      try {
        // A killed child can briefly remain a zombie until the container reaps it.
        running = !/\) Z /.test(readFileSync(`/proc/${pid}/stat`, 'utf8'));
      } catch (error) { if (error.code === 'ENOENT') running = false; else throw error; }
      if (running) await delay(10);
    }
    assert.equal(running, false, 'owned child remained running after forced cleanup');
  } finally { rmSync(dir, { recursive: true, force: true }); }
});
