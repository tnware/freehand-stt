import type { Page } from "@playwright/test";

declare global {
  interface Window {
    testOutput: {
      calls: string[];
      enabledInstances: () => string[];
      pendingEnables: number;
      failNextEnable: () => void;
      holdNextEnable: () => void;
      releaseEnable: () => void;
    };
  }
}

/** Mock bounded output reads; process actions keep their separate runtime fixture. */
export async function installOutputFixture(page: Page) {
  await page.route(
    "**/bindings/**/internal/managedruntime/manager.*",
    (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
      const enabled = new Set();
      const calls = [];
      let failEnable = false;
      let holdEnable = false;
      let releaseEnable;
      window.testOutput = {
        calls,
        enabledInstances: () => [...enabled],
        pendingEnables: 0,
        failNextEnable: () => { failEnable = true; },
        holdNextEnable: () => { holdEnable = true; },
        releaseEnable: () => releaseEnable?.(),
      };
      export const EnableProcessOutput = async ({instanceID}) => {
        calls.push('enable:' + instanceID);
        if (failEnable) { failEnable = false; throw new Error('Fixture enable failed'); }
        if (holdEnable) {
          holdEnable = false;
          window.testOutput.pendingEnables++;
          await new Promise(resolve => { releaseEnable = resolve; });
          releaseEnable = undefined;
          window.testOutput.pendingEnables--;
        }
        enabled.add(instanceID);
      };
      export const DisableProcessOutput = async ({instanceID}) => {
        enabled.delete(instanceID);
        calls.push('disable:' + instanceID);
      };
      export const ClearProcessOutput = async ({instanceID}) => { calls.push('clear:' + instanceID); };
      export const ReadProcessOutput = async ({instanceID}) => {
        calls.push('read:' + instanceID);
        if (!enabled.has(instanceID)) throw new Error('Read before enable');
        return { enabled: true, chunks: [{ sequence: 1, timestamp: 0, stream: 'stderr', text: '<b>Sensitive startup diagnostic</b> for ' + instanceID + '\\r\\n' }], next: 1, truncated: false };
      };
      `,
      }),
  );
}

export function outputCalls(page: Page) {
  return page.evaluate(() => window.testOutput.calls);
}
