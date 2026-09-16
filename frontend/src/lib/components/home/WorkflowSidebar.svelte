<script lang="ts">
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import { Switch } from "$lib/components/ui/switch";
  import VoiceTranscriptionSettings from "./VoiceTranscriptionSettings.svelte";
  import QuickSettings from "./QuickSettings.svelte";
  import SpeechQuickSettings from "./SpeechQuickSettings.svelte";
  import type {
    QuickSettingsPatch,
    QuickSettingsField,
  } from "$lib/stores/editor.svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import { FileTranscriptionPhase, State, TTSPhase } from "$lib/state";

  let {
    session,
    workflow,
    title,
    onAddConnection,
    onOpenOptions,
    onOpenCleanup,
    onOpenVocabulary,
    onOpenDelivery,
    onOpenOverlay,
    onOpenRuntime,
    disabled = false,
    editing = false,
  }: {
    session: Session;
    workflow: "voice" | "file" | "tts";
    title: string;
    onAddConnection: (purpose: Purpose) => void;
    onOpenOptions: () => void;
    onOpenCleanup: () => void;
    onOpenVocabulary: () => void;
    onOpenDelivery: () => void;
    onOpenOverlay: () => void;
    onOpenRuntime: () => void;
    disabled?: boolean;
    editing?: boolean;
  } = $props();

  const settings = $derived(session.editor.applied ?? session.editor.draft);
  const fileWorking = $derived(
    [
      FileTranscriptionPhase.FileTranscriptionUploading,
      FileTranscriptionPhase.FileTranscriptionProcessing,
      FileTranscriptionPhase.FileTranscriptionStreaming,
      FileTranscriptionPhase.FileTranscriptionCancelling,
    ].includes(session.files.status.phase),
  );
  const runtimeWorkBusy = $derived(
    ![State.Idle, State.Failed].includes(session.dictation.status.state) ||
      session.files.starting ||
      fileWorking ||
      session.speech.submitting ||
      session.speech.previewing ||
      session.speech.listening !== null ||
      session.speech.status.phase === TTSPhase.Generating,
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
  const controlsDisabled = $derived(
    disabled || editing || session.editor.dirty || session.editor.saving,
  );
  function update(patch: QuickSettingsPatch, field: QuickSettingsField) {
    if (controlsDisabled) return Promise.resolve(false);
    return session.editor.updateQuickSettings(patch, field);
  }
</script>

<!--
  Immediate workflow controls use the confirmed settings snapshot. The right
  pane owns detailed configuration and its unsaved draft.
-->
<aside
  class="workflow-settings flex w-[252px] shrink-0 flex-col border-r border-hairline bg-layer-fill"
  aria-label={`${title} settings`}
>
  <SidebarHeader {title} />

  <div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto px-1.5 py-2">
    {#if settings}
      <div
        class="flex min-w-0 flex-col gap-3 px-2.5 pb-1"
        role="group"
        aria-label={`${title} quick settings`}
      >
        {#if workflow === "voice"}
          <VoiceTranscriptionSettings
            sidebar
            {settings}
            editor={session.editor}
            runtime={session.runtime}
            {runtimeWorkBusy}
            disabled={controlsDisabled}
            {onAddConnection}
            onManageRuntime={onOpenRuntime}
            onOpenAdvanced={onOpenOptions}
          />
        {:else if workflow === "tts"}
          <SpeechQuickSettings
            sidebar
            {settings}
            editor={session.editor}
            runtime={session.runtime}
            {runtimeWorkBusy}
            disabled={controlsDisabled}
            {onAddConnection}
            onManageRuntime={onOpenRuntime}
            onOpenSettings={onOpenOptions}
          />
        {/if}
        {#if workflow !== "tts"}
          <QuickSettings
            sidebar
            embedded
            showTranscription={workflow === "file"}
            {settings}
            runtime={session.runtime}
            {runtimeWorkBusy}
            onManageRuntime={onOpenRuntime}
            processingProfiles={session.editor.processingProfiles}
            connection={session.editor.connection}
            processingConnection={session.editor.processingConnection}
            sttStale={session.editor.sttConnectionStale ||
              session.editor.connectionResultStale(
                Purpose.Transcription,
                settings,
              )}
            processingStale={session.editor.processingConnectionStale ||
              session.editor.connectionResultStale(Purpose.Cleanup, settings)}
            pending={session.editor.quickSettingsPending}
            savedField={session.editor.quickSettingsSaved}
            failedField={session.editor.quickSettingsFailed}
            sttTesting={session.editor.sttConnectionTesting}
            processingTesting={session.editor.processingConnectionTesting}
            onEnterTranscription={() =>
              void session.editor.ensureConnectionMetadata(
                Purpose.Transcription,
                true,
              )}
            onEnterCleanup={() =>
              void session.editor.ensureConnectionMetadata(
                Purpose.Cleanup,
                true,
              )}
            sttMetadataStatus={session.editor.connectionMetadataStatus(
              Purpose.Transcription,
            )}
            processingMetadataStatus={session.editor.connectionMetadataStatus(
              Purpose.Cleanup,
            )}
            {onAddConnection}
            onChangeConnection={(change) =>
              controlsDisabled
                ? Promise.resolve(false)
                : session.editor.changeConnection(change)}
            onUpdate={update}
            onTestConnection={() =>
              session.editor.testAppliedConnection(Purpose.Transcription)}
            onTestProcessingConnection={() =>
              session.editor.testAppliedConnection(Purpose.Cleanup)}
            onOpenServerSettings={onOpenOptions}
            onOpenProcessingSettings={onOpenCleanup}
            disabled={controlsDisabled}
          />
        {/if}
      </div>
      {#if workflow === "file"}
        <div class="srow">
          <span class="sk">Stream results</span>
          <Switch
            checked={session.files.streamingEnabled}
            onCheckedChange={(next) => {
              if (
                !controlsDisabled &&
                !session.files.selectionBusy &&
                !fileWorking
              )
                session.files.streamingPreferred = next;
            }}
            disabled={controlsDisabled ||
              session.files.selectionBusy ||
              fileWorking ||
              session.files.status.streamingUnavailable}
            aria-label="Stream partial results"
            class="scale-[0.8]"
          />
        </div>
      {/if}
      {#if workflow !== "tts"}
        <button
          type="button"
          class="srow"
          onclick={onOpenVocabulary}
          {disabled}
        >
          <span class="sk">Vocabulary</span>
          <span class="sv text-muted-foreground"
            >{vocabularyEnabled
              ? `${terms} ${terms === 1 ? "term" : "terms"}`
              : "Off for this workflow"}<ChevronRightIcon
              class="size-3 shrink-0"
            /></span
          >
        </button>
      {/if}
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
      <p class="px-2.5 pt-0.5 pb-1 text-xs leading-snug text-ink-quiet">
        {workflow === "tts"
          ? "Audio is generated on request and never saved automatically."
          : "A file transcript is never inserted into another application."}
      </p>
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
