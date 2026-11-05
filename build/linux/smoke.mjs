// Explicit, no-inference qualification of the installed frontend/Go bridge.
import { spawn } from 'node:child_process';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const readyMarker = 'FREEHAND_FRONTEND_BRIDGE_READY_V1';
const inherited = ['PATH', 'HOME', 'DISPLAY', 'XAUTHORITY', 'DBUS_SESSION_BUS_ADDRESS',
  'XDG_CONFIG_HOME', 'XDG_CACHE_HOME', 'XDG_DATA_HOME', 'XDG_STATE_HOME',
  'XDG_RUNTIME_DIR', 'XDG_SESSION_TYPE', 'GDK_BACKEND', 'LANG', 'LC_ALL'];

export function smokeEnvironment(source) {
  const result = Object.fromEntries(inherited.filter(key => source[key] !== undefined).map(key => [key, source[key]]));
  result.FREEHAND_LINUX_SHELL_SMOKE = '1';
  return result;
}

export function qualifyShell(binary, args = [], options = {}) {
  if (!binary.startsWith('/')) throw new Error('Use an absolute installed executable path');
  const timeoutMs = options.timeoutMs ?? 90000;
  const settleMs = options.settleMs ?? 500;
  const cleanupMs = options.cleanupMs ?? 1000;
  const maxOutput = options.maxOutput ?? 65536;
  return new Promise((resolveResult, reject) => {
    const child = spawn(binary, args, { env: smokeEnvironment(options.env ?? process.env), detached: true, stdio: ['ignore', 'pipe', 'pipe'] });
    let bytes = 0, pending = '', ready = false, finished = false, settle;
    let stderrTail = '', launchIssue = 'unknown', exitStatus = 'unknown';
    const killGroup = signal => {
      if (!child.pid) return;
      try { process.kill(-child.pid, signal); } catch (error) { if (error.code !== 'ESRCH') throw error; }
    };
    const deadline = setTimeout(() => finish(new Error('Frontend bridge readiness timed out')), timeoutMs);
    function finish(error) {
      if (finished) return;
      finished = true;
      clearTimeout(deadline); clearTimeout(settle);
      try { killGroup('SIGTERM'); } catch (cause) { error ??= cause; }
      // Always clean the owned group, even when the parent exited first. This
      // is bounded test cleanup, not proof of graceful application shutdown.
      setTimeout(() => {
        try { killGroup('SIGKILL'); } catch (cause) { error ??= cause; }
        if (error) {
          // Read only fixed classifications, after pipes have had cleanup time
          // to drain. Never publish raw app output, paths, settings or drafts.
          error.message += ` [launch=${launchIssue}, exit=${exitStatus}]`;
          stderrTail = '';
          reject(error);
        }
        else resolveResult({ frontendBridgeReady: true, cleanup: 'owned-process-group-terminated' });
      }, cleanupMs);
    }
    child.on('error', error => finish(error));
    child.on('exit', (code, signal) => {
      exitStatus = signal ?? (Number.isInteger(code) ? `code_${code}` : 'unknown');
      if (!finished) finish(new Error('Application exited before readiness stability check'));
    });
    function count(chunk) {
      bytes += chunk.length;
      if (bytes > maxOutput) finish(new Error('Application output exceeded the bounded smoke budget'));
    }
    child.stderr.on('data', chunk => {
      count(chunk);
      if (bytes > maxOutput) return;
      // Keep a small rolling match window, not the application log. A detected
      // namespace failure stays classified even if a later native stack trace
      // is longer than this window. This changes no launch or sandbox setting.
      const sample = stderrTail + chunk.toString('utf8');
      stderrTail = sample.slice(-2048);
      if (/bwrap:[^\n]*(?:No permissions to create a new namespace|Creating new namespace failed: Operation not permitted)/i.test(sample)) {
        launchIssue = 'namespace_denied';
      } else if (launchIssue === 'unknown' && /error while loading shared libraries:/i.test(sample)) {
        launchIssue = 'native_library_unavailable';
      } else if (launchIssue === 'unknown' && /cannot open display:|could not connect to display/i.test(sample)) {
        launchIssue = 'display_unavailable';
      }
    });
    child.stdout.on('data', chunk => {
      count(chunk);
      if (finished) return;
      pending += chunk.toString('utf8');
      const lines = pending.split('\n'); pending = lines.pop();
      for (const line of lines) {
        if (!ready && line.replace(/\r$/, '') === readyMarker) {
          ready = true;
          settle = setTimeout(() => finish(), settleMs);
        }
      }
    });
  });
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    await qualifyShell(process.argv[2] ?? '/usr/bin/freehand');
    console.log('PASS: installed frontend executed and called its Go bridge; initialization, hardware and graceful shutdown remain separate checks.');
  } catch (error) {
    console.error(`FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}
