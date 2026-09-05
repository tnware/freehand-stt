<script lang="ts">
  import ConnectionsSection from "$lib/components/settings/sections/ConnectionsSection.svelte";
  import SavedConnectionPicker from "$lib/components/settings/SavedConnectionPicker.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import { Button } from "$lib/components/ui/button";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import SettingsNav from "$lib/components/settings/SettingsNav.svelte";
  import AudioSection from "$lib/components/settings/sections/AudioSection.svelte";
  import GeneralSection from "$lib/components/settings/sections/GeneralSection.svelte";
  import HistorySection from "$lib/components/settings/sections/HistorySection.svelte";
  import OverlaySection from "$lib/components/settings/sections/OverlaySection.svelte";
  import ServerSection from "$lib/components/settings/sections/ServerSection.svelte";
  import ProcessingSection from "$lib/components/settings/sections/ProcessingSection.svelte";
  import ShortcutsSection from "$lib/components/settings/sections/ShortcutsSection.svelte";
  import SpeechSection from "$lib/components/settings/sections/SpeechSection.svelte";
  import { sectionByID } from "$lib/navigation";
  import type { SettingsSectionID } from "$lib/navigation";
  import type { Session } from "$lib/stores/session.svelte";
  import { State } from "$lib/state";
  import type { Message } from "$lib/utils/messages";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import { cn } from "$lib/utils";

  let {
    session,
    active = $bindable(),
    navigationRef = $bindable(null),
    onClose,
    overlayPreviewing,
    onStartOverlayPreview,
    onStopOverlayPreview,
  }: {
    session: Session;
    active: SettingsSectionID;
    navigationRef?: HTMLElement | null;
    onClose: () => void;
    overlayPreviewing: boolean;
    onStartOverlayPreview: () => void;
    onStopOverlayPreview: () => void;
  } = $props();

  const section = $derived(sectionByID(active));
  const dirty = $derived(session.editor.dirty);

  function selectSection(id: SettingsSectionID) {
    if (session.editor.connectionDraft && id !== "connections") {
      session.messages.reportInfo(
        "Save or cancel your connection edit before leaving Connections.",
      );
      return;
    }
    active = id;
    if (id === "audio") void session.editor.refreshDevices();
  }

  async function saveSettings() {
    if (await session.editor.save()) shortcutCapture.markSaved();
  }

  // Settings has no transport to state its own progress, so the channel carries
  // both outcomes of whatever you just pressed.
  const messages = $derived.by(() => {
    const out: Message[] = [];
    const configuration = session.editor.applied?.configuration;
    const preservedFields = configuration?.preservedFields ?? [];
    if (preservedFields.length > 0) {
      const remaining = Math.max(
        0,
        (configuration?.preservedFieldCount ?? preservedFields.length) - preservedFields.length,
      );
      out.push({
        id: "configuration-compatibility",
        tone: "info",
        source: "system",
        text: `Settings from a newer Freehand version are preserved but cannot be edited here: ${preservedFields.join(", ")}${remaining > 0 ? `, and ${remaining} more` : ""}.`,
      });
    }
    if (session.messages.info) {
      out.push({
        id: "system-info",
        tone: "info",
        source: "system",
        text: session.messages.info,
        onDismiss: () => session.messages.dismissInfo(),
      });
    }
    if (session.messages.error) {
      out.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.messages.error,
        onDismiss: () => session.messages.dismissError(),
      });
    }
    if (session.messages.notice) {
      out.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.messages.notice,
        onDismiss: () => session.messages.dismissNotice(),
      });
    }
    return out;
  });
</script>

<div class="flex min-h-0 flex-1">
  <SettingsNav {active} onSelect={selectSection} bind:navigationRef />

  <div class="flex min-w-0 flex-1 flex-col">
    <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5">
      <section aria-labelledby="settings-section-title" class="flex max-w-[620px] flex-col gap-3.5">
        <div class="flex items-baseline gap-2.5">
          <h3 class="text-base font-semibold tracking-[-0.01em]">
            {section.label}
          </h3>
          <p class="min-w-0 text-[12px] text-muted-foreground">
            {section.blurb}
          </p>
        </div>
        <h2 id="settings-section-title" class="sr-only" aria-live="polite" aria-atomic="true">
          {section.label} settings
        </h2>
        <Notifications {messages} />

        {#if session.editor.draft}
          {#if active === "server" || active === "processing" || active === "speech"}
            <SavedConnectionPicker
              catalog={session.editor.draft.savedConnections}
              purpose={active === "server"
                ? Purpose.Transcription
                : active === "processing"
                  ? Purpose.Cleanup
                  : Purpose.Speech}
              dirty={session.editor.dirty}
              busy={session.editor.saving || session.editor.quickSettingsPending.length > 0}
              onChange={(change) => session.editor.changeConnection(change)}
              onManage={() => {
                if (session.editor.runtimeDirty) {
                  session.messages.reportInfo(
                    "Save or discard feature settings before editing a connection.",
                  );
                  return;
                }
                const purpose =
                  active === "server"
                    ? Purpose.Transcription
                    : active === "processing"
                      ? Purpose.Cleanup
                      : Purpose.Speech;
                const c = session.editor.applied?.savedConnections.entries?.find(
                  (c) => c.id === session.editor.applied?.savedConnections.selected?.[purpose],
                );
                active = "connections";
                if (c) session.editor.beginConnection(c);
              }}
            />
          {/if}
          {#if active === "connections"}
            <ConnectionsSection
              editor={session.editor}
              error={session.messages.error}
              onOpenFeature={(p) =>
                selectSection(
                  p === Purpose.Transcription
                    ? "server"
                    : p === Purpose.Cleanup
                      ? "processing"
                      : "speech",
                )}
            />
          {:else if active === "general"}
            <GeneralSection bind:settings={session.editor.draft} />
          {:else if active === "shortcuts"}
            <ShortcutsSection
              bind:settings={session.editor.draft}
              status={session.dictation.status}
              busy={session.busy}
              capture={shortcutCapture}
            />
          {:else if active === "audio"}
            <AudioSection
              bind:settings={session.editor.draft}
              devices={session.editor.devices}
              microphoneChoice={session.editor.microphoneChoice}
              busy={session.editor.devicesBusy}
              onChooseMicrophone={(choice) => session.editor.chooseMicrophone(choice)}
              onRefreshDevices={() => session.editor.refreshDevices()}
            />
          {:else if active === "overlay"}
            <OverlaySection
              bind:settings={session.editor.draft}
              previewing={overlayPreviewing}
              canPreview={session.dictation.status.state === State.Idle ||
                session.dictation.status.state === State.Failed}
              onStartPreview={onStartOverlayPreview}
              onStopPreview={onStopOverlayPreview}
            />
          {:else if active === "server"}
            {#if session.editor.draft.savedConnections.selected?.stt}
              <ServerSection
                bind:settings={session.editor.draft}
                connection={session.editor.connection}
                busy={session.editor.sttConnectionTesting}
                onTestConnection={() => session.editor.testConnection()}
              />
            {/if}
          {:else if active === "processing"}
            {#if session.editor.draft.savedConnections.selected?.cleanup}
              <ProcessingSection
                bind:settings={session.editor.draft}
                profiles={session.editor.processingProfiles}
                connection={session.editor.processingConnection}
                busy={session.editor.processingConnectionTesting}
                onTestConnection={() => session.editor.testPostProcessingConnection()}
              />
            {/if}
          {:else if active === "speech"}
            {#if session.editor.draft.savedConnections.selected?.speech}
              <SpeechSection
                bind:settings={session.editor.draft}
                status={session.speech.status}
                busy={session.speech.previewing}
                connection={session.editor.ttsConnection}
                connectionBusy={session.editor.ttsConnectionTesting}
                canPreview={session.dictation.status.state === State.Idle &&
                  !session.files.status.canCancel}
                onPreview={() => session.speech.previewVoice()}
                onStop={() => session.speech.stopTTS()}
                onSave={() => session.speech.saveTTSAudio()}
                onClear={() => session.speech.clearTTSAudio()}
                onTestConnection={() => session.editor.testTextToSpeechConnection()}
              />
            {/if}
          {:else if active === "history"}
            <HistorySection
              bind:settings={session.editor.draft}
              enabled={session.editor.applied?.historyEnabled ?? false}
              entries={session.history.entries}
              onCopy={(id) => session.history.copyHistoryEntry(id)}
              onCopyVersion={(id, version) => session.history.copyHistoryEntryVersion(id, version)}
              onDelete={(id) => session.history.deleteHistoryEntry(id)}
              onClear={() => session.history.clearHistory()}
            />
          {/if}
        {:else}
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-24 w-full" />
        {/if}
      </section>
    </div>

    <div
      class="flex h-[58px] shrink-0 items-center justify-end gap-2.5 border-t border-hairline bg-layer-fill px-5"
    >
      {#if session.editor.draft}
        <span
          class="figure mr-auto flex items-center gap-2 text-[10.5px] text-muted-foreground"
          aria-live="polite"
          aria-atomic="true"
        >
          <span
            class={cn(
              "size-1.5 rounded-full",
              dirty || session.editor.saving ? "bg-primary" : "bg-success",
            )}
            aria-hidden="true"
          ></span>
          {session.editor.saving
            ? "Saving changes…"
            : dirty
              ? "Unsaved changes"
              : "All changes saved"}
        </span>
        <Button variant="outline" disabled={session.editor.saving} onclick={onClose}>Close</Button>
        {#if active !== "connections"}<Button
            disabled={session.busy || shortcutCapture.capturing || !dirty}
            onclick={saveSettings}
          >
            {#if session.editor.saving}
              <LoaderCircleIcon data-icon="inline-start" class="animate-spin" />
            {/if}
            {session.editor.saving
              ? "Saving…"
              : ["server", "processing", "speech"].includes(active)
                ? "Save feature settings"
                : "Save settings"}
          </Button>{/if}
      {:else}
        <Button variant="outline" disabled={session.editor.saving} onclick={onClose}>Close</Button>
      {/if}
    </div>
  </div>
</div>
