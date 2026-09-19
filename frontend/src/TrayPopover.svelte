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
  import { statusMessage, isCopyRequired } from "$lib/utils/status";
  import { CopyFeedback } from "$lib/utils/copyFeedback.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { appReadiness } from "$lib/utils/readiness";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import RuntimeStatus from "$lib/components/common/RuntimeStatus.svelte";
  import { Button } from "$lib/components/ui/button";
  import MicIcon from "@lucide/svelte/icons/mic";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import XIcon from "@lucide/svelte/icons/x";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import CheckIcon from "@lucide/svelte/icons/check";
  import ArrowUpRightIcon from "@lucide/svelte/icons/arrow-up-right";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import FileTextIcon from "@lucide/svelte/icons/file-text";

  let {
    session = new Session(),
    windows = WindowingService,
  }: {
    session?: Session;
    windows?: Pick<
      typeof WindowingService,
      "HideTrayPopover" | "OpenMain" | "OpenSettings"
    >;
  } = $props();
  let loading = $state(true),
    cancelling = $state(false),
    copying = $state(false);
  const feedback = new CopyFeedback();
  const settings = $derived(session.editor.applied);
  const status = $derived(session.dictation.status);
  const recording = $derived(status.state === State.Recording);
  const working = $derived(
    ![State.Idle, State.Failed, State.Recording].includes(status.state),
  );
  const connection = $derived(
    settings?.savedConnections.entries?.find(
      (entry) =>
        entry.id === settings.savedConnections.selected?.[Purpose.Voice],
    ),
  );
  const backend = $derived(
    settings?.compatibilityProfiles.transcription?.find(
      (profile) =>
        profile.id === settings.voiceTranscription.compatibilityProfile,
    ),
  );
  const serverLoaded = $derived(
    !!backend?.capabilities.serverLoadedModel &&
      !backend?.capabilities.realtime,
  );
  const instanceID = $derived(
    settings?.voiceTranscription.managedInstanceID ?? "",
  );
  const runtime = $derived(session.runtime.statusFor(instanceID));
  const local = $derived(
    runtimePresentation(
      runtime?.status,
      undefined,
      session.runtime.pendingFor(instanceID),
    ),
  );
  const model = $derived(
    instanceID
      ? local.selected?.name ||
          runtime?.instance.model ||
          "No local model selected"
      : serverLoaded
        ? "Server-loaded model"
        : settings?.voiceTranscription.model || "No model selected",
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
      (instanceID
        ? local.ready
        : !!connection &&
          !!settings.voiceTranscription.baseURL &&
          (!!settings.voiceTranscription.model || serverLoaded)),
  );
  const text = $derived(recording || working ? "" : (status.transcript ?? ""));
  const resultKey = $derived(String(status.generation));
  const error = $derived(
    session.messages.error ||
      (isCopyRequired(status) ? status.message || "" : statusMessage(status)),
  );
  const needsCopy = $derived(isCopyRequired(status));
  const shortcut = $derived(
    settings?.toggleShortcut || settings?.holdShortcut || "",
  );
  const shortcutLabel = $derived(
    settings?.toggleShortcut ? "Toggle recording" : "Hold to talk",
  );
  const phase = $derived(
    loading
      ? "Loading Freehand…"
      : recording
        ? "Recording in progress"
        : status.state === State.Transcribing
          ? "Transcribing audio"
          : status.state === State.PostProcessing
            ? "Cleaning up transcript"
            : status.state === State.Ready
              ? "Checking delivery"
              : status.state === State.Cancelling
                ? "Cancelling dictation"
                : needsCopy
                  ? "Transcript ready to copy"
                  : error
                    ? "Needs attention"
                    : !configured
                      ? "Voice setup needed"
                      : !shortcut
                        ? "Set a recording shortcut"
                        : "Ready to dictate",
  );
  const tone = $derived(
    recording
      ? "recording"
      : needsCopy
        ? "warning"
        : error
          ? "danger"
          : loading || working
            ? "accent"
            : !configured || !shortcut
              ? "warning"
              : "accent",
  );
  const guidance = $derived(
    recording
      ? "To finish, focus your original destination and use your recording shortcut."
      : working
        ? "Return to your original destination while processing finishes."
        : needsCopy
          ? "Copy the result below. Check your destination for any text already inserted before pasting."
          : !configured
            ? "Open Voice settings to finish setup or resolve the connection."
            : !shortcut
              ? "Assign a shortcut to dictate into the app you’re using."
              : "Focus a text field in your app, then use your shortcut.",
  );
  const canCopy = $derived(
    !loading && !recording && !working && (!!text || status.canCopy),
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
  async function open(action: () => Promise<void>) {
    try {
      await action();
    } catch (cause) {
      session.messages.fail(cause);
    }
  }
  function voiceSettings() {
    void open(() => windows.OpenSettings("voice-transcription"));
  }
  async function cancel() {
    if (
      loading ||
      cancelling ||
      !status.canCancel ||
      status.state === State.Cancelling
    )
      return;
    cancelling = true;
    try {
      await session.dictation.cancel();
    } finally {
      cancelling = false;
    }
  }
  async function copy() {
    if (copying || !canCopy) return;
    copying = true;
    try {
      await feedback.copy(resultKey, () => session.dictation.copyCurrent());
    } finally {
      copying = false;
    }
  }
</script>

<ModeWatcher defaultMode="system" disableTransitions />
<main class="tray" aria-label="Freehand quick view" aria-busy={loading}>
  <header class="tray-header">
    <div class="flex min-w-0 items-center gap-2">
      <BrandMark /><span class="font-display text-sm font-semibold"
        >Freehand</span
      ><span class="text-xs text-muted-foreground">Quick view</span>
    </div>
    <div class="flex shrink-0 items-center gap-1">
      <Button
        variant="ghost"
        size="icon-xs"
        aria-label="Voice settings"
        title="Voice settings"
        onclick={voiceSettings}><SettingsIcon /></Button
      >
      <Button
        variant="ghost"
        size="icon-xs"
        aria-label="Close quick view"
        title="Close quick view"
        onclick={() => open(() => windows.HideTrayPopover())}><XIcon /></Button
      >
    </div>
  </header>
  <div class="tray-body">
    <section class="activity" data-tone={tone} aria-label="Dictation status">
      <div class="flex items-start gap-3">
        <span class="activity-icon" aria-hidden="true">
          {#if loading || working}<LoaderCircleIcon
              class="size-5 motion-safe:animate-spin"
            />
          {:else if tone === "warning" || tone === "danger"}<CircleAlertIcon
              class="size-5"
            />
          {:else}<MicIcon class="size-5" />{/if}
        </span>
        <div class="min-w-0 flex-1">
          <h1 class="text-[14px] font-semibold leading-5" role="status">
            {phase}
          </h1>
          {#if !loading}<p
              class="mt-1 text-xs leading-relaxed text-secondary-foreground"
            >
              {guidance}
            </p>{/if}
        </div>
      </div>
      {#if !loading && !working && !needsCopy}
        {#if shortcut}
          <div class="shortcut-row">
            <span class="text-xs text-muted-foreground">{shortcutLabel}</span
            ><ShortcutKeys
              value={shortcut}
              platform={settings?.platform}
              label={shortcutLabel}
            />
          </div>
        {:else}<Button
            variant="outline"
            size="sm"
            class="mt-3 w-full"
            onclick={() => open(() => windows.OpenSettings("shortcuts"))}
            >Configure shortcut<ChevronRightIcon /></Button
          >{/if}
      {/if}
      {#if !loading && !configured && !recording && !working}
        <Button
          variant="soft"
          size="sm"
          class="mt-3 w-full"
          onclick={voiceSettings}
          >Open Voice settings<ChevronRightIcon /></Button
        >
      {/if}
      {#if status.canCancel && (recording || working)}
        <Button
          variant="outline"
          size="sm"
          class="mt-3 w-full"
          disabled={loading || cancelling || status.state === State.Cancelling}
          onclick={cancel}
          ><XIcon />{cancelling || status.state === State.Cancelling
            ? "Cancelling…"
            : recording
              ? "Cancel recording"
              : "Cancel dictation"}</Button
        >
      {/if}
    </section>
    {#if error}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (Long error details support keyboard scrolling.) -->
      <div
        class="tray-error"
        class:recovery={needsCopy && !session.messages.error}
        role="alert"
        tabindex="0"
      >
        {error}
      </div>
    {/if}
    {#if settings}
      <section aria-label="Voice configuration" class="voice-summary">
        <button
          type="button"
          class="voice-link"
          onclick={voiceSettings}
          aria-label="Open Voice configuration"
        >
          <ProviderIcon
            profile={settings.voiceTranscription.compatibilityProfile}
            size={20}
          />
          <span class="min-w-0 flex-1 text-left"
            ><span class="block text-[13px] font-medium break-words"
              >{connection?.name || "No Voice connection"}</span
            ><span
              class="mt-0.5 block truncate text-xs text-muted-foreground"
              title={model}>{model}</span
            ></span
          >
          <ChevronRightIcon
            class="size-3.5 shrink-0 text-muted-foreground"
            aria-hidden="true"
          />
        </button>
        <div
          class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 px-2 pb-2 text-xs text-muted-foreground"
        >
          {#if instanceID}<RuntimeStatus view={local} /><span aria-hidden="true"
              >·</span
            >{/if}
          <span
            >{settings.voiceTranscription.realtime
              ? "Live dictation"
              : "Completed transcription"}</span
          >
          {#if instanceID}<button
              class="runtime-link"
              onclick={() => open(() => windows.OpenSettings("local-runtime"))}
              >Manage runtime</button
            >{/if}
        </div>
      </section>
    {/if}
    <section class="result" aria-label="Latest dictation">
      <div class="flex items-center justify-between gap-2">
        <h2
          class="flex items-center gap-1.5 text-xs font-medium text-secondary-foreground"
        >
          <FileTextIcon class="size-3.5" aria-hidden="true" />Latest dictation
        </h2>
        <Button
          variant={needsCopy ? "soft" : "ghost"}
          size="xs"
          disabled={copying || !canCopy}
          onclick={copy}
        >
          {#if feedback.key === resultKey}<CheckIcon />Copied{:else}<CopyIcon
            />Copy{/if}
        </Button>
      </div>
      {#if text}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex (Scrollable transcript supports keyboard navigation.) -->
        <div
          class="transcript"
          role="region"
          aria-label="Latest transcript"
          tabindex="0"
        >
          {text}
        </div>
      {:else}<div class="result-empty">
          <FileTextIcon
            class="size-5 text-muted-foreground"
            aria-hidden="true"
          />
          <p>
            {recording
              ? "Listening for speech…"
              : working
                ? "Your transcript will appear when ready."
                : "Your next dictation will appear here."}
          </p>
        </div>{/if}
    </section>
  </div>
  <footer class="tray-footer">
    <Button
      variant="outline"
      size="sm"
      class="w-full justify-between"
      onclick={() => open(() => windows.OpenMain())}
      >Open Freehand<ArrowUpRightIcon /></Button
    >
  </footer>
</main>

<style>
  .tray {
    height: 100dvh;
    width: 100%;
    min-width: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--background);
    color: var(--foreground);
  }
  .tray-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--hairline);
    background: var(--layer-fill);
    flex-shrink: 0;
  }
  .tray-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 12px;
  }
  .activity {
    flex-shrink: 0;
    padding: 12px;
    border: 1px solid var(--accent-edge);
    border-radius: var(--radius-md);
    background: var(--accent-wash);
  }
  .activity-icon {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    color: var(--accent-text);
  }
  .activity[data-tone="recording"] {
    border-color: var(--record-edge);
    background: var(--record-wash);
  }
  .activity[data-tone="recording"] .activity-icon {
    color: var(--record-text);
  }
  .activity[data-tone="warning"] {
    border-color: color-mix(in srgb, var(--warning) 40%, var(--hairline));
    background: color-mix(in srgb, var(--warning) 6%, var(--background));
  }
  .activity[data-tone="warning"] .activity-icon {
    color: var(--warning);
  }
  .activity[data-tone="danger"] {
    border-color: color-mix(in srgb, var(--destructive) 40%, var(--hairline));
    background: color-mix(in srgb, var(--destructive) 6%, var(--background));
  }
  .activity[data-tone="danger"] .activity-icon {
    color: var(--destructive);
  }
  .shortcut-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
    margin-top: 12px;
    padding-top: 10px;
    border-top: 1px solid var(--hairline);
  }
  .voice-summary {
    min-width: 0;
    flex-shrink: 0;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-md);
    background: var(--well);
  }
  .voice-link {
    display: flex;
    width: 100%;
    min-width: 0;
    align-items: center;
    gap: 8px;
    padding: 8px;
    border-radius: var(--radius-md);
  }
  .voice-link:hover {
    background: var(--subtle-fill-hover);
  }
  .runtime-link {
    color: var(--accent-text);
    margin-left: auto;
  }
  .runtime-link:hover {
    text-decoration: underline;
  }
  .voice-link:focus-visible,
  .runtime-link:focus-visible,
  .transcript:focus-visible,
  .tray-error:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }
  .tray-error {
    flex-shrink: 0;
    max-height: 80px;
    overflow: auto;
    overflow-wrap: anywhere;
    border-left: 2px solid var(--destructive);
    padding-left: 8px;
    font-size: 12px;
    line-height: 1.5;
    color: var(--destructive);
  }
  .tray-error.recovery {
    border-color: var(--warning);
    color: var(--warning);
  }
  .result {
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1 0 112px;
    min-width: 0;
  }
  .transcript {
    min-height: 60px;
    max-height: 140px;
    overflow: auto;
    overflow-wrap: anywhere;
    white-space: pre-wrap;
    padding: 8px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-md);
    background: var(--well);
    font-size: 12px;
    line-height: 1.65;
  }
  .result-empty {
    flex: 1;
    min-height: 72px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    border: 1px dashed var(--hairline);
    border-radius: var(--radius-md);
    text-align: center;
    font-size: 12px;
    line-height: 1.5;
    color: var(--muted-foreground);
  }
  .tray-footer {
    flex-shrink: 0;
    padding: 10px 12px;
    border-top: 1px solid var(--hairline);
    background: var(--layer-fill);
  }
</style>
