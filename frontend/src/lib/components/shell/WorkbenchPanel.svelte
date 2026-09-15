<script lang="ts">
  import { Purpose } from "$bindings/savedconnection";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { FileTranscriptionPhase, State } from "$lib/state";
  import HistoryPanel from "$lib/components/home/HistoryPanel.svelte";
  import RuntimeOutputDrawer from "$lib/components/runtimes/RuntimeOutputDrawer.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import PanelTabs from "./PanelTabs.svelte";
  import RuntimeOutputTabs from "./RuntimeOutputTabs.svelte";

  let {
    session,
    inputMode,
    settingsOpen,
    onOpenHistorySettings,
  }: {
    session: Session;
    inputMode: string;
    settingsOpen: boolean;
    onOpenHistorySettings: () => void;
  } = $props();

  const layout = getWorkbenchLayout();
  const uid = $props.id();
  const panelID = `${uid}-content`;
  const outputPanelID = `${uid}-runtime-output`;
  const settings = $derived(session.editor.applied);
  const tabs = $derived([
    {
      id: "recent",
      label: settings?.historyEnabled
        ? `Recent · ${session.history.entries.length}`
        : "Recent",
    },
    { id: "output", label: "Runtime output" },
    { id: "diagnostics", label: "Diagnostics" },
  ]);
  const instances = $derived(
    [...session.runtime.instances].sort(
      (left, right) =>
        left.instance.name.localeCompare(right.instance.name) ||
        left.instance.id.localeCompare(right.instance.id),
    ),
  );
  const outputRow = $derived(
    instances.find((row) => row.instance.id === layout?.outputInstanceID),
  );

  // The initial output target is independent of page navigation. Once chosen,
  // even removal leaves the target explicit until the user selects another.
  $effect(() => {
    if (!layout || layout.tab !== "output" || layout.outputInstanceID) return;
    const initial =
      instances.find((row) => row.status.state === "running") ?? instances[0];
    if (initial) layout.outputInstanceID = initial.instance.id;
  });

  function selectTab(value: string) {
    if (
      layout &&
      (value === "recent" || value === "output" || value === "diagnostics")
    )
      layout.tab = value;
  }

  function selectRuntime(id: string) {
    if (layout && session.runtime.statusFor(id)) layout.outputInstanceID = id;
  }

  const diagnosticWorkflow = $derived(layout?.diagnosticWorkflow ?? "voice");
  const diagnosticPurpose = $derived(
    diagnosticWorkflow === "file" ? Purpose.Transcription : Purpose.Voice,
  );
  const diagnosticResult = $derived(
    session.editor.connectionMetadataResult(diagnosticPurpose),
  );
  const diagnosticBusy = $derived(
    session.editor.connectionMetadataBusy(diagnosticPurpose),
  );
  const diagnosticStatus = $derived(
    session.editor.connectionMetadataStatus(diagnosticPurpose),
  );
  const hasConnection = $derived(
    Boolean(settings?.savedConnections.selected?.[diagnosticPurpose]),
  );
  const diagnosticsLocked = $derived(
    settingsOpen || session.editor.saving || !hasConnection || diagnosticBusy,
  );
  function checkConnection() {
    if (!diagnosticsLocked)
      void session.editor.testAppliedConnection(diagnosticPurpose);
  }

  const voiceActive = $derived(
    session.dictation.status.state !== State.Idle &&
      session.dictation.status.state !== State.Failed,
  );
  const fileWorking = $derived(
    session.files.starting ||
      [
        FileTranscriptionPhase.FileTranscriptionUploading,
        FileTranscriptionPhase.FileTranscriptionProcessing,
        FileTranscriptionPhase.FileTranscriptionStreaming,
        FileTranscriptionPhase.FileTranscriptionCancelling,
      ].includes(session.files.status.phase),
  );
</script>

{#if layout}
  <section
    class="flex min-h-0 min-w-0 flex-1 flex-col"
    aria-label="Workbench panel"
  >
    <PanelTabs
      {tabs}
      bind:active={() => layout.tab, selectTab}
      bind:collapsed={
        () => false,
        (collapsed) => {
          if (collapsed) layout.toggleBottom();
        }
      }
      {panelID}
      label="Workbench panel"
    />
    <div
      id={panelID}
      role="tabpanel"
      aria-label={tabs.find((tab) => tab.id === layout.tab)?.label}
      class="flex min-h-0 min-w-0 flex-1 flex-col"
    >
      {#if layout.tab === "recent"}
        <HistoryPanel
          enabled={settings?.historyEnabled ?? false}
          entries={session.history.entries}
          fileStatus={session.files.status}
          fileHistoryGeneration={session.files.historyGeneration}
          onOpenSettings={onOpenHistorySettings}
          onCopy={(id) => session.history.copyHistoryEntry(id)}
          onCopyVersion={(id, version) =>
            session.history.copyHistoryEntryVersion(id, version)}
          onDelete={(id) => session.history.deleteHistoryEntry(id)}
          onCopyFile={() => session.files.copyFileTranscript()}
          ttsEnabled={settings?.textToSpeech.enabled ?? false}
          ttsAvailable={!voiceActive &&
            !fileWorking &&
            session.speech.canListen}
          ttsPending={session.speech.listening ?? undefined}
          ttsStatus={session.speech.status}
          onListen={(id, version) =>
            session.speech.listenHistoryEntry(id, version)}
          onListenFile={() => session.speech.listenFileTranscript()}
          onPauseTTS={() => session.speech.pauseTTS()}
          onResumeTTS={() => session.speech.resumeTTS()}
          onRestartTTS={() => session.speech.restartTTS()}
          onSeekTTS={(request) => session.speech.seekTTS(request)}
          seeking={session.speech.seeking}
          saving={session.speech.saving}
          onStopTTS={() => session.speech.stopTTS()}
          onSaveTTS={() => session.speech.saveTTSAudio()}
          onClearTTS={() => session.speech.clearTTSAudio()}
          ttsWorkspaceVisible={inputMode === "tts"}
          collapsible={false}
          playbackVisibleElsewhere
        />
      {:else if layout.tab === "output"}
        <RuntimeOutputTabs
          {instances}
          selected={layout.outputInstanceID}
          panelID={outputPanelID}
          onSelect={selectRuntime}
        />
        <div
          id={outputPanelID}
          role="tabpanel"
          aria-label={outputRow?.instance.name ?? "Runtime output selection"}
          class="flex min-h-0 min-w-0 flex-1 flex-col"
        >
          {#if outputRow}
            {#key outputRow.instance.id}<RuntimeOutputDrawer
                instanceID={outputRow.instance.id}
              />{/key}
          {:else}
            <p
              class="flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-5 py-3 text-center text-xs text-muted-foreground"
            >
              {layout.outputInstanceID
                ? "The selected runtime is no longer available. Choose another runtime to inspect its output."
                : "Install a local runtime to inspect its output."}
            </p>
          {/if}
        </div>
      {:else}
        <div
          class="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-5 py-1.5"
        >
          <label for={`${uid}-workflow`} class="text-xs text-muted-foreground"
            >Workflow</label
          >
          <Select.Root
            type="single"
            value={diagnosticWorkflow}
            onValueChange={(value) => {
              if (value === "voice" || value === "file")
                layout.diagnosticWorkflow = value;
            }}
          >
            <Select.Trigger
              id={`${uid}-workflow`}
              size="sm"
              class="max-w-full min-w-0 w-48"
            >
              {diagnosticWorkflow === "voice"
                ? "Voice transcription"
                : "Audio file"}
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="voice" label="Voice transcription"
                >Voice transcription</Select.Item
              >
              <Select.Item value="file" label="Audio file"
                >Audio file</Select.Item
              >
            </Select.Content>
          </Select.Root>
          <Button
            variant="outline"
            size="xs"
            disabled={diagnosticsLocked}
            onclick={checkConnection}
          >
            {diagnosticBusy
              ? "Checking…"
              : diagnosticResult || diagnosticStatus === "failed"
                ? "Check again"
                : "Check connection"}
          </Button>
        </div>
        <div
          class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-3"
        >
          {#if settingsOpen}<p class="mb-3 text-xs text-muted-foreground">
              Leave Settings to check the saved connection.
            </p>{/if}
          {#if diagnosticResult}
            <ConnectionDiagnostics
              result={diagnosticResult}
              platform={settings?.platform ?? "windows"}
              busy={diagnosticBusy}
            />
          {:else}
            <p class="text-[13px] text-muted-foreground" role="status">
              {!hasConnection
                ? "Choose a connection for this workflow in Settings."
                : diagnosticBusy
                  ? "Checking the saved connection…"
                  : diagnosticStatus === "failed"
                    ? "The connection check did not complete. Check again to retry."
                    : "No current endpoint check is available for this workflow."}
            </p>
            <p class="mt-2 text-xs text-muted-foreground">
              Checks read server metadata using saved settings. They do not
              invoke a model.
            </p>
          {/if}
        </div>
      {/if}
    </div>
  </section>
{/if}
