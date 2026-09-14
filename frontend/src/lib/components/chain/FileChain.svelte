<script lang="ts">
  import Chain from "./Chain.svelte";
  import FileSourceStage from "./FileSourceStage.svelte";
  import TranscribeStage from "./TranscribeStage.svelte";
  import CleanupStage from "./CleanupStage.svelte";
  import DeliverStage from "./DeliverStage.svelte";
  import type { StageTone } from "./Stage.svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { endpointHost } from "$lib/utils/endpoint";
  import { taskConnectionStatus } from "$lib/utils/connection";
  import { FileTranscriptionPhase } from "$lib/state";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";

  let {
    session,
    now,
    blocked = "",
    transcribeModel,
    transcribeSource,
    onOpenTranscribe,
    onOpenCleanup,
    onOpenDelivery,
  }: {
    session: Session;
    now: number;
    /** Why a file cannot be started right now, empty when it can. */
    blocked?: string;
    transcribeModel: string;
    transcribeSource: string;
    onOpenTranscribe: () => void;
    onOpenCleanup: () => void;
    onOpenDelivery: () => void;
  } = $props();

  const status = $derived(session.files.status);
  const settings = $derived(session.editor.applied ?? session.editor.draft);

  const connection = $derived(taskConnectionStatus("file", session.editor, now));
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
    status.phase === FileTranscriptionPhase.FileTranscriptionStreaming,
  );
  const working = $derived(
    streaming ||
      status.phase === FileTranscriptionPhase.FileTranscriptionUploading ||
      status.phase === FileTranscriptionPhase.FileTranscriptionProcessing,
  );
  const failed = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionFailed,
  );
  const completed = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionCompleted,
  );

  const cleanup = $derived(settings?.postProcessing);

  function toggleCleanup(next: boolean) {
    void session.editor.updateQuickSettings(
      { postProcessing: { enabled: next } },
      "processing-enabled",
    );
  }
</script>

<Chain
  label="Audio file chain"
  links={[
    { label: "audio", tone: working ? "live" : "idle" },
    { label: "text", tone: working ? "live" : "idle" },
    { label: "text", tone: "idle" },
  ]}
  stages={[source, transcribe, clean, deliver]}
/>

{#snippet source()}
  <FileSourceStage
    {status}
    {blocked}
    choosing={session.files.choosing}
    starting={session.files.starting}
    cancelling={session.files.cancelling}
    clearing={session.files.clearing}
    onChoose={() => session.files.chooseAudioFile()}
    onStart={() => session.files.startFileTranscription()}
    onCancel={() => session.files.cancelFileTranscription()}
    onClear={() => session.files.clearAudioFile()}
  />
{/snippet}

{#snippet transcribe()}
  <TranscribeStage
    model={transcribeModel}
    source={transcribeSource}
    tone={failed ? "bad" : working ? "busy" : transcribeTone}
    state={streaming ? "Streaming" : connection.label}
    live={streaming ? (status.transcript ?? "") : ""}
    notice={status.streamingNotice ?? ""}
    onOpen={onOpenTranscribe}
  >
    {#snippet extra()}
      {#if status.streamingUnavailable && !working}
        <Button
          size="xs"
          variant="outline"
          onclick={() => session.files.tryFileStreamingAgain()}
          disabled={session.files.resettingStreaming}>Try streaming again</Button
        >
      {:else if !working}
        <span class="flex items-center gap-1.5">
          <Switch
            checked={session.files.streamingEnabled}
            onCheckedChange={(next) =>
              (session.files.streamingPreferred = next)}
            aria-label="Stream partial results"
            class="scale-[0.8]"
          />
          <span class="text-[11px] text-muted-foreground">Stream</span>
        </span>
      {/if}
    {/snippet}
  </TranscribeStage>
{/snippet}

{#snippet clean()}
  <CleanupStage
    enabled={!!cleanup?.enabled}
    model={cleanup?.model ?? ""}
    source={cleanup?.baseURL ? endpointHost(cleanup.baseURL) : ""}
    profile={cleanup?.compatibilityProfile ?? ""}
    busy={session.editor.isQuickSettingsPending("processing-enabled")}
    onToggle={toggleCleanup}
    onOpen={onOpenCleanup}
  />
{/snippet}

{#snippet deliver()}
  <!-- A file transcript is never inserted anywhere: this workflow has no
       captured target window, so the only delivery is one the user asks for. -->
  <DeliverStage
    ordinal="04"
    label="Deliver"
    pickLabel="Delivery"
    target="Copy or save"
    meta="never inserted automatically"
    tone={failed ? "bad" : completed ? "ok" : "idle"}
    notice={failed ? (status.message ?? "") : ""}
    noticeTitle="Transcription failed"
    canCopy={status.canCopy}
    onCopy={() => void session.files.copyFileTranscript()}
    onOpen={onOpenDelivery}
  />
{/snippet}
