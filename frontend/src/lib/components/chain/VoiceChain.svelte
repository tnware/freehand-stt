<script lang="ts">
  import Chain from "./Chain.svelte";
  import InputStage from "./InputStage.svelte";
  import TranscribeStage from "./TranscribeStage.svelte";
  import CleanupStage from "./CleanupStage.svelte";
  import DeliverStage from "./DeliverStage.svelte";
  import type { StageTone } from "./Stage.svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { endpointHost } from "$lib/utils/endpoint";
  import { taskConnectionStatus } from "$lib/utils/connection";
  import { isCopyRequired, isFailure, statusMessage } from "$lib/utils/status";
  import { State } from "$lib/state";
  import { Purpose } from "$bindings/savedconnection";
  import VoiceTranscriptionSettings from "$lib/components/home/VoiceTranscriptionSettings.svelte";
  import QuickSettings from "$lib/components/home/QuickSettings.svelte";

  let {
    session,
    now,
    busy = false,
    microphone = "System default",
    availability = "",
    transcribeModel,
    transcribeSource,
    onOpenAudio,
    onOpenTranscribe,
    onOpenCleanup,
    onOpenDelivery,
    onOpenRuntimes,
    onAddConnection,
  }: {
    session: Session;
    now: number;
    busy?: boolean;
    microphone?: string;
    availability?: string;
    /** Resolved by the caller, which already knows whether a managed runtime
     *  is serving this workflow. */
    transcribeModel: string;
    transcribeSource: string;
    onOpenAudio: () => void;
    onOpenTranscribe: () => void;
    onOpenCleanup: () => void;
    onOpenDelivery: () => void;
    onOpenRuntimes: () => void;
    onAddConnection: (purpose: Purpose) => void;
  } = $props();

  const status = $derived(session.dictation.status);
  const settings = $derived(session.editor.applied ?? session.editor.draft);

  const connection = $derived(
    taskConnectionStatus("voice", session.editor, now),
  );
  const transcribeTone = $derived<StageTone>(
    connection.dot === "bg-success"
      ? "ok"
      : connection.dot === "bg-warning"
        ? "warn"
        : connection.dot === "bg-destructive"
          ? "bad"
          : "idle",
  );

  const streaming = $derived(
    status.live &&
      (status.state === State.Recording ||
        status.state === State.Transcribing),
  );

  const cleanup = $derived(settings?.postProcessing);
  const cleanupBusy = $derived(
    session.editor.isQuickSettingsPending("processing-enabled") ||
      status.state === State.PostProcessing,
  );

  const waiting = $derived(isCopyRequired(status));
  const failed = $derived(isFailure(status));
  const deliverTone = $derived<StageTone>(
    failed ? "bad" : waiting ? "warn" : "ok",
  );

  function toggleCleanup(next: boolean) {
    void session.editor.updateQuickSettings(
      { postProcessing: { enabled: next } },
      "processing-enabled",
    );
  }
</script>

<Chain
  label="Voice transcription chain"
  links={[
    { label: "audio", tone: streaming ? "live" : "idle" },
    { label: "text", tone: streaming ? "live" : "idle" },
    { label: "text", tone: waiting ? "blocked" : "idle" },
  ]}
  stages={[input, transcribe, clean, deliver]}
/>

{#snippet input()}
  <InputStage
    {status}
    {now}
    {busy}
    {microphone}
    {availability}
    toggleShortcut={settings?.toggleShortcut ?? ""}
    onToggle={() => session.dictation.toggleRecording()}
    onCancel={() => session.dictation.cancel()}
    onOpenAudioSettings={onOpenAudio}
  />
{/snippet}

{#snippet transcribe()}
  <TranscribeStage
    model={transcribeModel}
    source={transcribeSource}
    tone={transcribeTone}
    state={connection.label}
    live={streaming ? (status.liveFinal ?? "") : ""}
    partial={streaming ? (status.livePartial ?? "") : ""}
    onOpen={onOpenTranscribe}
    panel={settings ? transcribePanel : undefined}
  />
{/snippet}

{#snippet clean()}
  <CleanupStage
    enabled={!!cleanup?.enabled}
    model={cleanup?.model ?? ""}
    source={cleanup?.baseURL ? endpointHost(cleanup.baseURL) : ""}
    profile={cleanup?.compatibilityProfile ?? ""}
    busy={cleanupBusy}
    onToggle={toggleCleanup}
    onOpen={onOpenCleanup}
    panel={settings ? cleanupPanel : undefined}
  />
{/snippet}

{#snippet transcribePanel()}
  <div class="p-3">
    <VoiceTranscriptionSettings
      runtime={session.runtime}
      onManageRuntime={onOpenRuntimes}
      editor={session.editor}
      settings={settings!}
      disabled={session.editor.saving}
      {onAddConnection}
    />
  </div>
{/snippet}

{#snippet cleanupPanel()}
  <div class="p-3">
    <QuickSettings
      embedded
      showCapture={false}
      showTranscription={false}
      settings={settings!}
      runtime={session.runtime}
      onManageRuntime={onOpenRuntimes}
      devices={session.editor.devices}
      processingProfiles={session.editor.processingProfiles}
      connection={session.editor.connection}
      processingConnection={session.editor.processingConnection}
      processingStale={session.editor.processingConnectionStale ||
        session.editor.connectionResultStale(Purpose.Cleanup, settings)}
      pending={session.editor.quickSettingsPending}
      savedField={session.editor.quickSettingsSaved}
      failedField={session.editor.quickSettingsFailed}
      processingTesting={session.editor.processingConnectionTesting}
      processingMetadataStatus={session.editor.connectionMetadataStatus(
        Purpose.Cleanup,
      )}
      onEnterCleanup={() =>
        void session.editor.ensureConnectionMetadata(Purpose.Cleanup, true)}
      onUpdate={(patch, field) =>
        session.editor.updateQuickSettings(patch, field)}
      onChangeConnection={(change) => session.editor.changeConnection(change)}
      onTestConnection={() =>
        session.editor.testConnection(session.editor.applied, "")}
      onTestProcessingConnection={() =>
        session.editor.testPostProcessingConnection(session.editor.applied, "")}
      onAddConnection={onAddConnection}
      onOpenServerSettings={onOpenCleanup}
      onOpenProcessingSettings={onOpenCleanup}
      onOpenAudioSettings={onOpenCleanup}
      onOpenDeliverySettings={onOpenCleanup}
      disabled={session.editor.saving}
    />
  </div>
{/snippet}


{#snippet deliver()}
  <DeliverStage
    target={settings?.autoInsert ? "Focused app" : "Copy only"}
    meta={settings?.autoInsert
      ? "insert · copy fallback"
      : "never inserted automatically"}
    tone={deliverTone}
    notice={waiting || failed ? (statusMessage(status) ?? "") : ""}
    noticeTitle={failed ? "Delivery failed" : "Copy required"}
    canCopy={status.canCopy}
    onCopy={() => void session.dictation.copyPending()}
    onOpen={onOpenDelivery}
  />
{/snippet}
