import { render } from "svelte/server";
import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it } from "vitest";
import type { Status } from "$bindings/managedruntime";
import { Session, idle, serviceWithStatus } from "$lib/stores/session-fixtures";
import ResultQuickSettings from "$lib/components/home/ResultQuickSettings.svelte";
import TrayPopover from "../TrayPopover.svelte";
import HomeScreen from "$lib/components/home/HomeScreen.svelte";
import SettingsScreen from "$lib/components/settings/SettingsScreen.svelte";

const runtime: Status = {
  supported: true,
  enabled: true,
  state: "running",
  selectedModel: "nemotron-local",
  realtime: true,
  backend: "cpu",
  version: "0.1.0",
  progress: -1,
  phase: "",
  error: "",
  models: [
    {
      id: "nemotron-local",
      name: "Nemotron 3.5",
      description: "",
      sizeBytes: 0,
      installed: true,
      recommended: true,
      realtime: true,
      profile: "nemotron",
    },
  ],
};
const noop = () => {};
async function managedSession(status: Status | null = runtime) {
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle)),
  );
  await session.editor.load();
  session.editor.applied!.managedRuntime = {
    enabled: true,
    model: "nemotron-local",
    realtime: true,
  };
  session.editor.applied!.voiceTranscription.model = "saved-manual-model";
  session.runtime.status = status;
  return session;
}

describe("managed speech surfaces", () => {
  it.each(["voice", "file"])(
    "shows managed readiness, not manual setup, in the %s task",
    async (inputMode) => {
      const session = await managedSession({ ...runtime, state: "stopped" });
      const { body } = render(HomeScreen, {
        props: {
          session,
          inputMode,
          onOpenHistorySettings: noop,
          onOpenServerSettings: noop,
          onOpenProcessingSettings: noop,
          onOpenAudioSettings: noop,
          onOpenShortcutSettings: noop,
          onOpenSpeechSettings: noop,
          onOpenGeneralSettings: noop,
        },
      });
      expect(body.includes("Nemotron 3.5")).toBe(true);
      expect(body.includes("Manage local runtime")).toBe(true);
      expect(body.includes('id="voice-connection"')).toBe(false);
      expect(body.includes("saved-manual-model")).toBe(false);
      session.dispose();
    },
  );

  it.each(["voice-transcription", "server"] as const)(
    "labels saved %s settings as inactive when runtime status is unavailable",
    async (active) => {
      const session = await managedSession(null);
      const { body } = render(SettingsScreen, {
        props: {
          session,
          active,
          onClose: noop,
          overlayPreviewing: false,
          onStartOverlayPreview: noop,
          onStopOverlayPreview: noop,
        },
      });
      expect(body.includes("inactive")).toBe(true);
      expect(body.includes("Manage local runtime")).toBe(true);
      expect(body.includes('aria-label="Active connection"')).toBe(false);
      session.dispose();
    },
  );

  it.each([true, false])(
    "identifies the local model in quick settings (Voice=%s)",
    async (showCapture) => {
      const session = await managedSession();
      const { body } = render(ResultQuickSettings, {
        props: {
          editor: session.editor,
          settings: session.editor.applied!,
          runtimeState: session.runtime,
          showCapture,
          disabled: false,
          onAddConnection: noop,
          onOpenLocalRuntime: noop,
          onOpenServerSettings: noop,
          onOpenProcessingSettings: noop,
          onOpenAudioSettings: noop,
          onOpenGeneralSettings: noop,
        },
      });
      expect(body).toContain("Nemotron 3.5");
      expect(body).not.toContain("saved-manual-model");
      session.dispose();
    },
  );

  it.each([runtime, null])(
    "does not expose manual tray model switches while managed (%j)",
    async (status) => {
      const session = await managedSession(status);
      const { body } = render(TrayPopover, { props: { session } });
      expect(body).toContain(status ? "Nemotron 3.5" : "nemotron-local");
      expect(body).toContain("Manage local runtime");
      expect(body).not.toContain('id="tray-connection"');
      expect(body).not.toContain('id="tray-model"');
      expect(body).not.toContain("saved-manual-model");
      if (!status) expect(body).toContain("Status unavailable");
      session.dispose();
    },
  );
});
