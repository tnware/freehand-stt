<script lang="ts">
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
  const dirty = $derived(session.settingsDirty);

  function selectSection(id: SettingsSectionID) {
    active = id;
    if (id === "audio") void session.refreshDevices();
  }

  async function saveSettings() {
    if (await session.save()) shortcutCapture.markSaved();
  }

  // Settings has no transport to state its own progress, so the channel carries
  // both outcomes of whatever you just pressed.
  const messages = $derived.by(() => {
    const out: Message[] = [];
    const configuration = session.appliedSettings?.configuration;
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
    if (session.info) {
      out.push({
        id: "system-info",
        tone: "info",
        source: "system",
        text: session.info,
        onDismiss: () => session.dismissInfo(),
      });
    }
    if (session.error) {
      out.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.error,
        onDismiss: () => session.dismissError(),
      });
    }
    if (session.notice) {
      out.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.notice,
        onDismiss: () => session.dismissNotice(),
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
          <h3 class="text-base font-semibold tracking-[-0.01em]">{section.label}</h3>
          <p class="min-w-0 text-[12px] text-muted-foreground">{section.blurb}</p>
        </div>
        <h2
          id="settings-section-title"
          class="sr-only"
          aria-live="polite"
          aria-atomic="true"
        >
          {section.label} settings
        </h2>
        <Notifications {messages} />

        {#if session.settings}
          {#if active === "general"}
            <GeneralSection bind:settings={session.settings} />
          {:else if active === "shortcuts"}
            <ShortcutsSection
              bind:settings={session.settings}
              status={session.status}
              busy={session.busy}
              capture={shortcutCapture}
            />
          {:else if active === "audio"}
            <AudioSection
              bind:settings={session.settings}
              devices={session.devices}
              microphoneChoice={session.microphoneChoice}
              busy={session.devicesBusy}
              onChooseMicrophone={(choice) => session.chooseMicrophone(choice)}
              onRefreshDevices={() => session.refreshDevices()}
            />
          {:else if active === "overlay"}
            <OverlaySection
              bind:settings={session.settings}
              previewing={overlayPreviewing}
              canPreview={session.status.state === State.Idle || session.status.state === State.Failed}
              onStartPreview={onStartOverlayPreview}
              onStopPreview={onStopOverlayPreview}
            />
          {:else if active === "server"}
            <ServerSection
              bind:settings={session.settings}
              bind:apiKey={session.apiKey}
              bind:clearKey={session.clearKey}
              connection={session.connection}
              busy={session.sttConnectionTesting}
              onTestConnection={() => session.testConnection()}
            />
          {:else if active === "processing"}
            <ProcessingSection
              bind:settings={session.settings}
              bind:apiKey={session.processingAPIKey}
              bind:clearKey={session.clearProcessingKey}
              profiles={session.processingProfiles}
              connection={session.processingConnection}
              busy={session.processingConnectionTesting}
              onTestConnection={() => session.testPostProcessingConnection()}
            />
          {:else if active === "speech"}
            <SpeechSection
              bind:settings={session.settings}
              bind:apiKey={session.ttsAPIKey}
              bind:clearKey={session.clearTTSKey}
              status={session.ttsStatus}
              busy={session.ttsPreviewing}
              connection={session.ttsConnection}
              connectionBusy={session.ttsConnectionTesting}
              canPreview={
                session.status.state === State.Idle &&
                !session.fileStatus.canCancel
              }
              onPreview={() => session.previewVoice()}
              onStop={() => session.stopTTS()}
              onSave={() => session.saveTTSAudio()}
              onClear={() => session.clearTTSAudio()}
              onTestConnection={() => session.testTextToSpeechConnection()}
            />
          {:else if active === "history"}
            <HistorySection
              bind:settings={session.settings}
              enabled={session.appliedSettings?.historyEnabled ?? false}
              entries={session.history}
              onCopy={(id) => session.copyHistoryEntry(id)}
              onCopyVersion={(id, version) => session.copyHistoryEntryVersion(id, version)}
              onDelete={(id) => session.deleteHistoryEntry(id)}
              onClear={() => session.clearHistory()}
            />
          {/if}
        {:else}
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-24 w-full" />
        {/if}
      </section>
    </div>

    <div class="flex h-[58px] shrink-0 items-center justify-end gap-2.5 border-t border-hairline bg-layer-fill px-5">
      {#if session.settings}
        <span
          class="figure mr-auto flex items-center gap-2 text-[10.5px] text-muted-foreground"
          aria-live="polite"
          aria-atomic="true"
        >
          <span
            class={cn(
              "size-1.5 rounded-full",
              dirty || session.settingsSaving ? "bg-primary" : "bg-success",
            )}
            aria-hidden="true"
          ></span>
          {session.settingsSaving
            ? "Saving changes…"
            : dirty
              ? "Unsaved changes"
              : "All changes saved"}
        </span>
        <Button variant="outline" disabled={session.settingsSaving} onclick={onClose}>Close</Button>
        <Button
          disabled={session.busy || shortcutCapture.capturing || !dirty}
          onclick={saveSettings}
        >
          {#if session.settingsSaving}
            <LoaderCircleIcon data-icon="inline-start" class="animate-spin" />
          {/if}
          {session.settingsSaving ? "Saving…" : "Save changes"}
        </Button>
      {:else}
        <Button variant="outline" disabled={session.settingsSaving} onclick={onClose}>Close</Button>
      {/if}
    </div>
  </div>
</div>
