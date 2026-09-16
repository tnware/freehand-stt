<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as WindowingService from "$bindings/windowing/service";
  import { Session } from "$lib/stores/session.svelte";
  import { subscribeSessionEvents } from "$lib/stores/session-events";
  import { activeAppearanceMode } from "$lib/appearance";
  import { State } from "$lib/state";
  import { Purpose } from "$bindings/savedconnection";
  import { canToggleRecording, statusMessage } from "$lib/utils/status";
  import { hideThenToggle } from "$lib/tray-actions";
  import { CopyFeedback } from "$lib/utils/copyFeedback.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { appReadiness } from "$lib/utils/readiness";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import QuickSaveStatus from "$lib/components/settings/QuickSaveStatus.svelte";
  import { Button } from "$lib/components/ui/button";
  import MicIcon from "@lucide/svelte/icons/mic";
  import SquareIcon from "@lucide/svelte/icons/square";

  let {
    session = new Session(),
    windows = WindowingService,
  }: {
    session?: Session;
    windows?: Pick<
      typeof WindowingService,
      "HideTrayPopover" | "OpenMain" | "OpenSettings" | "OpenTaskConnection"
    >;
  } = $props();
  let loading = $state(true),
    acting = $state(false),
    copying = $state(false);
  const feedback = new CopyFeedback();
  const settings = $derived(session.editor.applied);
  const status = $derived(session.dictation.status);
  const recording = $derived(status.state === State.Recording);
  const working = $derived(
    ![State.Idle, State.Failed, State.Recording].includes(status.state),
  );
  const connectionID = $derived(
    settings?.savedConnections.selected?.[Purpose.Voice] ?? "",
  );
  const backend = $derived(
    settings?.compatibilityProfiles.transcription?.find(
      (p) => p.id === settings.voiceTranscription.compatibilityProfile,
    ),
  );
  const serverLoaded = $derived(
    !!backend?.capabilities.serverLoadedModel &&
      !backend?.capabilities.realtime,
  );
  const instanceID = $derived(
    settings?.voiceTranscription.managedInstanceID ?? "",
  );
  const managed = $derived(!!instanceID);
  const runtime = $derived(session.runtime.statusFor(instanceID));
  const local = $derived(runtimePresentation(runtime?.status));
  const localModel = $derived(
    local.selected?.name ||
      runtime?.instance.model ||
      "No local model selected",
  );
  const readiness = $derived(
    settings
      ? appReadiness(
          settings,
          session.editor.currentVoiceConnection,
          session.editor.devices,
          session.editor.devicesBusy,
          "voice",
          runtime,
        )
      : null,
  );
  const configured = $derived(
    !!settings?.setupCompleted &&
      !!readiness &&
      !readiness.recoveryNeeded &&
      (managed
        ? local.ready
        : !!connectionID &&
          !!settings.voiceTranscription.baseURL &&
          (!!settings.voiceTranscription.model || serverLoaded)),
  );
  const busy = $derived(
    loading ||
      acting ||
      session.busy ||
      session.editor.quickSettingsPending.length > 0,
  );
  const locked = $derived(busy || recording || working);
  const text = $derived(status.transcript ?? "");
  const resultKey = $derived(String(status.generation));
  const error = $derived(session.messages.error || statusMessage(status));
  const phase = $derived(
    loading
      ? "Loading…"
      : recording
        ? "Recording"
        : status.canCopy
          ? "Ready to copy"
          : working
            ? status.state === State.Ready
              ? "Checking focus…"
              : "Working…"
            : error
              ? "Needs attention"
              : !configured
                ? "Setup needed"
                : "Ready to dictate",
  );
  $effect(() => {
    const mode = activeAppearanceMode(settings);
    untrack(() => setMode(mode));
  });
  onMount(() => {
    let alive = true;
    const off = subscribeSessionEvents(session, Events.On);
    void session.load().finally(() => {
      if (alive) loading = false;
    });
    return () => {
      alive = false;
      off();
      feedback.dispose();
      session.dispose();
    };
  });
  async function toggle() {
    if (
      busy ||
      !canToggleRecording(status, false) ||
      (!recording && !configured)
    )
      return;
    acting = true;
    try {
      await hideThenToggle(
        () => windows.HideTrayPopover(),
        () => session.dictation.toggleRecording(),
      );
    } catch (cause) {
      session.messages.fail(cause);
    } finally {
      acting = false;
    }
  }
  async function open(action: () => Promise<void>) {
    try {
      await action();
    } catch (cause) {
      session.messages.fail(cause);
    }
  }
  function settingsWindow() {
    void open(() =>
      windows.OpenSettings(managed ? "local-runtime" : "voice-transcription"),
    );
  }
  function connectionWindow(create: boolean) {
    void open(() =>
      windows.OpenTaskConnection(
        { id: "", purpose: Purpose.Voice, create },
        "voice",
      ),
    );
  }
  async function copy() {
    if (copying) return;
    copying = true;
    try {
      await feedback.copy(resultKey, () => session.dictation.copyCurrent());
    } finally {
      copying = false;
    }
  }
</script>

<ModeWatcher defaultMode="system" disableTransitions />
<main class="tray" aria-label="Freehand dictation" aria-busy={loading}>
  <header class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <BrandMark /><span
        class="font-display text-base font-semibold tracking-tight"
        >Freehand</span
      >
    </div>
    <span class="caption text-muted-foreground">VOICE</span>
  </header>
  <section class="capture" aria-label="Recording controls">
    <div
      class="flex items-center gap-2 text-xs text-muted-foreground"
      role="status"
    >
      <span class="dot" class:live={recording}></span>{phase}
    </div>
    <button
      class="record"
      class:live={recording}
      disabled={busy ||
        !canToggleRecording(status, false) ||
        (!recording && !configured)}
      onclick={toggle}
      aria-label={recording ? "Stop recording" : "Start recording"}
    >
      {#if recording}<SquareIcon
          class="size-5"
          fill="currentColor"
        />{:else}<MicIcon class="size-5" />{/if}
      {recording ? "Stop recording" : "Start recording"}
    </button>
    <p class="text-center text-xs leading-relaxed text-muted-foreground">
      {recording
        ? "Close this panel to keep recording."
        : "Hides this panel before recording."}
    </p>
  </section>
  {#if settings}
    <section class="configuration space-y-2.5" aria-label="Voice configuration">
      <div class="flex items-center justify-between">
        <label for="tray-connection" class="text-xs font-medium"
          >Voice connection</label
        ><button
          class="text-xs text-muted-foreground hover:text-foreground"
          onclick={settingsWindow}>Options ↗</button
        >
      </div>
      <ConnectionSelect
        id="tray-connection"
        catalog={settings.savedConnections}
        purpose={Purpose.Voice}
        compact
        disabled={locked}
        onChange={(change) => session.editor.changeConnection(change)}
        onAdd={() => connectionWindow(true)}
        onManage={() => connectionWindow(false)}
      />
      {#if managed}
        <p class="text-xs font-medium">Local transcription</p>
        <p class="truncate text-sm" title={localModel}>{localModel}</p>
        <p class="text-xs text-muted-foreground">
          {local.label} · {settings.voiceTranscription.realtime
            ? "Realtime"
            : "Completed transcription"}
        </p>
        <Button variant="outline" size="sm" onclick={settingsWindow}
          >Manage local runtime</Button
        >
      {:else}
        <RuntimeModelPicker
          id="tray-model"
          value={settings.voiceTranscription.model}
          compact
          immediate
          showProfileName={false}
          models={session.editor.currentVoiceConnection?.modelIDs ?? []}
          savedModels={rememberedModels(settings, Purpose.Voice).map(
            (m) => m.model,
          )}
          {serverLoaded}
          disabled={locked || !connectionID}
          busy={session.editor.voiceConnectionTesting}
          metadataStatus={session.editor.connectionMetadataStatus(
            Purpose.Voice,
          )}
          onEnter={() => {
            void session.editor.ensureConnectionMetadata(Purpose.Voice, true);
          }}
          onDiscover={() => {
            void session.editor.testAppliedConnection(Purpose.Voice);
          }}
          onChoose={(model) =>
            session.editor.updateQuickSettings(
              { voiceTranscription: { model } },
              "voice-transcription",
            )}
        />
        <QuickSaveStatus
          fields={["voice-transcription"]}
          pending={session.editor.quickSettingsPending}
          saved={session.editor.quickSettingsSaved}
          failed={session.editor.quickSettingsFailed}
        />
      {/if}
    </section>
  {/if}
  {#if error}<p class="error text-xs text-destructive" role="alert">
      {error}
    </p>{:else if !loading && !configured}<p
      class="text-xs text-muted-foreground"
    >
      Finish Voice setup in Settings to record.
    </p>{/if}
  <section class="result" aria-label="Latest result">
    <div class="flex items-center justify-between">
      <h2 class="caption">LATEST RESULT</h2>
      <Button
        variant="ghost"
        size="sm"
        disabled={copying || working || recording || (!text && !status.canCopy)}
        onclick={copy}>{feedback.key === resultKey ? "Copied" : "Copy"}</Button
      >
    </div>
    <p class="transcript text-xs leading-relaxed text-secondary-foreground">
      {text ||
        (recording
          ? "Listening…"
          : working
            ? "Your transcript is on its way…"
            : "Your latest transcript will appear here.")}
    </p>
  </section>
  <footer
    class="flex items-center justify-between border-t border-hairline pt-2"
  >
    <Button
      variant="ghost"
      size="sm"
      onclick={() => open(() => windows.OpenMain())}>Open Freehand ↗</Button
    ><Button variant="ghost" size="sm" onclick={settingsWindow}>Settings</Button
    >
  </footer>
</main>

<style>
  .tray {
    height: 100dvh;
    width: 100%;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
    padding: 14px 18px 8px;
    background: var(--background);
    color: var(--foreground);
  }
  .capture {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    background: var(--card);
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--muted-foreground);
  }
  .dot.live {
    background: var(--record);
  }
  .record {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 44px;
    border-radius: var(--radius-md);
    background: var(--primary);
    color: var(--primary-foreground);
    font-size: 14px;
    font-weight: 600;
  }
  .record.live {
    background: var(--record);
    color: white;
  }
  .record:disabled {
    opacity: 0.45;
  }
  button:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 3px;
  }
  .configuration {
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
  }
  .result {
    flex: 1;
    min-height: 55px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  .transcript {
    overflow: auto;
    overflow-wrap: anywhere;
    white-space: pre-wrap;
    min-height: 0;
  }
  .error {
    max-height: 54px;
    overflow: auto;
    overflow-wrap: anywhere;
    flex-shrink: 0;
  }
  header,
  .capture,
  footer {
    flex-shrink: 0;
  }
</style>
