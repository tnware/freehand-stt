<script lang="ts">
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import { Switch } from "$lib/components/ui/switch";
  import { Button } from "$lib/components/ui/button";
  import { endpointHost } from "$lib/utils/endpoint";
  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
  import type { Session } from "$lib/stores/session.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import { FileTranscriptionPhase, State } from "$lib/state";

  let {
    session,
    workflow,
    title,
    instanceID = "",
    onOpenConnection,
    onOpenModel,
    onOpenOptions,
    onOpenCleanup,
    onOpenVocabulary,
    onOpenDelivery,
    onOpenOverlay,
    onOpenRuntime,
    disabled = false,
  }: {
    session: Session;
    workflow: "voice" | "file" | "tts";
    title: string;
    /** The managed runtime serving this workflow, when there is one. */
    instanceID?: string;
    onOpenConnection: () => void;
    onOpenModel: () => void;
    onOpenOptions: () => void;
    onOpenCleanup: () => void;
    onOpenVocabulary: () => void;
    onOpenDelivery: () => void;
    onOpenOverlay: () => void;
    onOpenRuntime: () => void;
    disabled?: boolean;
  } = $props();

  const settings = $derived(session.editor.applied ?? session.editor.draft);
  const speech = $derived(settings?.textToSpeech);
  const cleanup = $derived(settings?.postProcessing);

  // Each workflow keeps its own endpoint and model; voice and speech nest
  // theirs, the file workflow uses the root fields.
  const endpoint = $derived(
    workflow === "voice"
      ? settings?.voiceTranscription
      : workflow === "tts"
        ? settings?.textToSpeech
        : settings,
  );
  const purpose = $derived(
    workflow === "voice"
      ? Purpose.Voice
      : workflow === "tts"
        ? Purpose.Speech
        : Purpose.Transcription,
  );
  const connection = $derived(
    settings?.savedConnections.entries?.find(
      (entry) => entry.id === settings.savedConnections.selected?.[purpose],
    ),
  );
  const runtime = $derived(
    instanceID ? session.runtime.statusFor(instanceID) : undefined,
  );
  const local = $derived(runtimePresentation(runtime?.status));
  const running = $derived(runtime?.status.state === "running");
  const operating = $derived(session.runtime.isBusy(instanceID));
  const selectedModel = $derived(
    runtime?.status.models?.find((item) => item.id === runtime.instance.model),
  );
  const voiceProfile = $derived(
    instanceID
      ? (selectedModel?.behavior ??
          settings?.modelProfiles.voiceTranscription?.find(
            (item) => item.id === settings.voiceTranscription.modelProfile,
          ))
      : settings?.modelProfiles.voiceTranscription?.find(
          (item) => item.id === settings.voiceTranscription.modelProfile,
        ),
  );
  const voiceBackend = $derived(
    settings?.compatibilityProfiles.transcription?.find(
      (item) => item.id === settings.voiceTranscription.compatibilityProfile,
    ),
  );
  const realtimeSupported = $derived(
    !!voiceProfile?.capabilities.realtime &&
      !!voiceBackend?.capabilities.realtime,
  );
  const fileWorking = $derived(
    [
      FileTranscriptionPhase.FileTranscriptionUploading,
      FileTranscriptionPhase.FileTranscriptionProcessing,
      FileTranscriptionPhase.FileTranscriptionStreaming,
      FileTranscriptionPhase.FileTranscriptionCancelling,
    ].includes(session.files.status.phase),
  );
  const runtimeLocked = $derived(
    disabled ||
      session.runtime.loading ||
      !runtime?.status.supported ||
      (session.dictation.status.state !== State.Idle &&
        session.dictation.status.state !== State.Failed) ||
      session.files.starting ||
      fileWorking,
  );
  const terms = $derived(
    (settings?.vocabulary.terms ?? "")
      .split(/\r?\n/)
      .map((term) => term.trim())
      .filter(Boolean).length,
  );
  const vocabularyEnabled = $derived(
    workflow === "voice"
      ? settings?.vocabulary.voice
      : settings?.vocabulary.files,
  );
  const model = $derived(
    instanceID
      ? local.selected?.name || runtime?.instance.model || "Local speech"
      : endpoint?.model || "Not selected",
  );

  function toggleRealtime(next: boolean) {
    if (
      disabled ||
      operating ||
      session.editor.isQuickSettingsPending("voice-transcription") ||
      (next && !realtimeSupported)
    )
      return;
    void session.editor.updateQuickSettings(
      { voiceTranscription: { realtime: next } },
      "voice-transcription",
    );
  }
</script>

<!--
  The workflow's settings, beside its work rather than behind a window. Every
  row is the current value and a way to change it, so the pane answers "what
  will happen when I run this" without going anywhere.
-->
<aside
  class="workflow-settings flex w-[252px] shrink-0 flex-col border-r border-hairline bg-layer-fill"
  aria-label={`${title} settings`}
>
  <SidebarHeader {title} />

  <div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto px-1.5 py-2">
    <button type="button" class="srow" onclick={onOpenConnection} {disabled}>
      <span class="sk">Connection</span>
      <span class="sv"
        >{instanceID
          ? runtime?.instance.name || "Local runtime"
          : connection?.name || "Not selected"}<ChevronRightIcon
          class="size-3 shrink-0 text-muted-foreground"
        /></span
      >
    </button>
    {#if endpoint?.baseURL}
      <p
        class="-mt-0.5 truncate px-2.5 pb-1 font-mono text-[10px] text-ink-quiet"
      >
        {endpointHost(endpoint.baseURL)}
      </p>
    {/if}

    <button type="button" class="srow" onclick={onOpenModel} {disabled}>
      <span class="sk">Model</span>
      <span class="sv font-mono text-[11px]">{model}</span>
    </button>

    {#if workflow === "tts"}
      <button type="button" class="srow" onclick={onOpenOptions} {disabled}>
        <span class="sk">Voice</span>
        <span class="sv">{speech?.voice || "Default"}</span>
      </button>
      <button type="button" class="srow" onclick={onOpenOptions} {disabled}>
        <span class="sk">Speed</span>
        <span class="sv">{(speech?.speed ?? 1).toFixed(2)}×</span>
      </button>
    {:else}
      <button type="button" class="srow" onclick={onOpenOptions} {disabled}>
        <span class="sk">Language</span>
        <span class="sv"
          >{(workflow === "voice"
            ? settings?.voiceTranscription.language
            : settings?.language) || "Server default"}</span
        >
      </button>

      {#if workflow === "voice" && (realtimeSupported || settings?.voiceTranscription.realtime)}
        <div class="srow">
          <span class="sk">Live dictation</span>
          <Switch
            checked={!!settings?.voiceTranscription.realtime}
            onCheckedChange={toggleRealtime}
            disabled={disabled ||
              operating ||
              session.editor.isQuickSettingsPending("voice-transcription") ||
              (!settings?.voiceTranscription.realtime &&
                (!realtimeSupported ||
                  (!instanceID && (!connection || !endpoint?.model))))}
            aria-label="Live dictation"
            class="scale-[0.8]"
          />
        </div>
      {:else if workflow === "file"}
        <div class="srow">
          <span class="sk">Stream results</span>
          <Switch
            id="file-stream-toggle"
            checked={session.files.streamingEnabled}
            onCheckedChange={(next) =>
              (session.files.streamingPreferred = next)}
            disabled={disabled ||
              session.files.selectionBusy ||
              fileWorking ||
              session.files.status.streamingUnavailable}
            aria-label="Stream partial results"
            class="scale-[0.8]"
          />
        </div>
      {/if}

      <button type="button" class="srow" onclick={onOpenCleanup} {disabled}>
        <span class="sk">Cleanup</span>
        <span class="sv"
          >{cleanup?.enabled ? cleanup.model || "On" : "Off"}<ChevronRightIcon
            class="size-3 shrink-0 text-muted-foreground"
          /></span
        >
      </button>

      <button type="button" class="srow" onclick={onOpenVocabulary} {disabled}>
        <span class="sk">Vocabulary</span>
        <span class="sv text-muted-foreground"
          >{vocabularyEnabled
            ? `${terms} ${terms === 1 ? "term" : "terms"} · supported models`
            : "Off for this workflow"}</span
        >
      </button>
    {/if}

    <div class="mx-2.5 my-2 h-px bg-hairline"></div>
    <p
      class="px-2.5 pb-1 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
    >
      Delivery
    </p>
    {#if workflow === "voice"}
      <button type="button" class="srow" onclick={onOpenDelivery} {disabled}>
        <span class="sk">Insert into</span>
        <span class="sv"
          >{settings?.autoInsert ? "Focused app" : "Copy only"}</span
        >
      </button>
      <button type="button" class="srow" onclick={onOpenDelivery} {disabled}>
        <span class="sk">If blocked</span>
        <span class="sv text-warning">Copy required</span>
      </button>
      <button type="button" class="srow" onclick={onOpenOverlay} {disabled}>
        <span class="sk">Overlay</span>
        <span class="sv">{settings?.overlayEnabled ? "On" : "Off"}</span>
      </button>
    {:else}
      <div class="srow">
        <span class="sk">Result</span>
        <span class="sv"
          >{workflow === "tts" ? "Play or save" : "Explicit copy"}</span
        >
      </div>
      <p class="px-2.5 pt-0.5 pb-1 text-[11px] leading-snug text-ink-quiet">
        {workflow === "tts"
          ? "Audio is generated on request and never saved automatically."
          : "A file transcript is never inserted into another application."}
      </p>
    {/if}

    {#if instanceID}
      <div class="mx-2.5 my-2 h-px bg-hairline"></div>
      <p
        class="px-2.5 pb-1.5 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
      >
        Local runtime
      </p>
      <div
        class="mx-1.5 rounded-md border border-hairline bg-subtle-fill-hover px-2.5 py-2"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="flex min-w-0 items-center gap-1.5">
            <span
              class="size-[7px] shrink-0 rounded-full {running
                ? 'bg-success'
                : runtime?.status.state === 'error'
                  ? 'bg-destructive'
                  : 'bg-meter-rest'}"
              aria-hidden="true"
            ></span>
            <span class="truncate text-[11.5px] text-secondary-foreground"
              >{runtime?.instance.name || "Local runtime"}</span
            >
          </span>
          {#if operating}
            <Button
              variant="outline"
              size="xs"
              disabled={disabled ||
                session.runtime.pendingFor(instanceID) === "Cancelling"}
              onclick={() => void session.runtime.cancel(instanceID)}
              >Cancel</Button
            >
          {:else}<Button
              variant="outline"
              size="xs"
              disabled={runtimeLocked ||
                (!running && (!local.installed || !selectedModel?.installed))}
              onclick={() =>
                void session.runtime.run(
                  instanceID,
                  running ? "Stop" : "Start",
                )}>{running ? "Stop" : "Start"}</Button
            >{/if}
        </div>
        <p class="mt-1.5 truncate font-mono text-[10px] text-ink-quiet">
          {[
            runtime?.status.version,
            runtime?.status.backend ? backendLabel(runtime.status.backend) : "",
            local.label,
          ]
            .filter(Boolean)
            .join(" · ")}
        </p>
        <button
          type="button"
          class="mt-1.5 text-[11px] text-accent-text underline-offset-2 hover:underline"
          onclick={onOpenRuntime}>Manage runtime</button
        >
      </div>
    {/if}
  </div>
</aside>

<style>
  /* One row shape for the whole pane: label left, current value right, the
     whole row a target. */
  .srow {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    min-height: 28px;
    flex-shrink: 0;
    padding: 0 0.5rem 0 0.625rem;
    border-radius: 5px;
    text-align: left;
    transition: background-color 100ms;
  }
  button.srow:hover:not(:disabled) {
    background: var(--subtle-fill-hover);
  }
  button.srow:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }
  button.srow:disabled {
    opacity: 0.6;
  }
  .srow :global(.sk) {
    flex: none;
    font-size: 11.5px;
    color: var(--muted-foreground);
  }
  .srow :global(.sv) {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 0.25rem;
    overflow: hidden;
    font-size: 11.5px;
    color: var(--secondary-foreground);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  @media (max-width: 699px) {
    .workflow-settings {
      width: 180px;
    }
    .srow {
      align-items: flex-start;
      flex-direction: column;
      gap: 2px;
      padding-block: 5px;
    }
    .srow :global(.sv) {
      max-width: 100%;
    }
  }
</style>
