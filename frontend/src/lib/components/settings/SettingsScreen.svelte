<script lang="ts">
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import { SETTINGS_NAVIGATION } from "$lib/navigation";
  import VocabularySection from "./sections/VocabularySection.svelte";
  import { setContext, tick, untrack } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import PendingChangesDialog from "./PendingChangesDialog.svelte";
  import * as WindowingService from "$bindings/windowing/service";
  import SavedConnectionPicker from "$lib/components/settings/SavedConnectionPicker.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import { Button } from "$lib/components/ui/button";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import SettingsNav from "$lib/components/settings/SettingsNav.svelte";
  import VoiceTranscriptionSettings from "$lib/components/home/VoiceTranscriptionSettings.svelte";
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
  import { State, FileTranscriptionPhase } from "$lib/state";
  import type { Message } from "$lib/utils/messages";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";

  let {
    session = $bindable(),
    visible = true,
    active = $bindable(),
    navigationRef = $bindable(null),
    onClose,
    onOpenRuntimes = onClose,
    onSaved = () => {},
    saveReturnsToTask = false,
    decisionOpen = false,
    onNavigate,
    onOpenConnection = (request) => {
      void WindowingService.OpenConnectionManager(request).catch((cause) =>
        session.messages.fail(cause),
      );
    },
    overlayPreviewing,
    onStartOverlayPreview,
    onStopOverlayPreview,
  }: {
    session: Session;
    visible?: boolean;
    active: SettingsSectionID;
    navigationRef?: HTMLElement | null;
    onClose: () => void;
    /** Leaves configuration for the runtime pane on the rail. */
    onOpenRuntimes?: () => void;
    onSaved?: () => void;
    saveReturnsToTask?: boolean;
    decisionOpen?: boolean;
    onNavigate?: (section: SettingsSectionID) => void;
    onOpenConnection?: (
      request: import("$bindings/windowing").ConnectionManagerRequest,
    ) => void;
    overlayPreviewing: boolean;
    onStartOverlayPreview: () => void;
    onStopOverlayPreview: () => void;
  } = $props();

  const section = $derived(sectionByID(active));
  const dirty = $derived(session.editor.dirty);
  const speechWorkBusy = $derived(
    ![State.Idle, State.Failed].includes(session.dictation.status.state) ||
      session.files.starting ||
      [
        FileTranscriptionPhase.FileTranscriptionUploading,
        FileTranscriptionPhase.FileTranscriptionProcessing,
        FileTranscriptionPhase.FileTranscriptionStreaming,
        FileTranscriptionPhase.FileTranscriptionCancelling,
      ].includes(session.files.status.phase),
  );
  const workflowPurpose = $derived(
    active === "voice-transcription"
      ? Purpose.Voice
      : active === "server"
        ? Purpose.Transcription
        : active === "processing"
          ? Purpose.Cleanup
          : active === "speech"
            ? Purpose.Speech
            : null,
  );
  $effect(() => {
    const purpose = workflowPurpose;
    const id =
      purpose && session.editor.applied?.savedConnections.selected?.[purpose];
    const checking = purpose && session.editor.connectionMetadataBusy(purpose);
    if (visible && purpose && id && !checking)
      untrack(() => {
        void session.editor.ensureConnectionMetadata(purpose);
      });
  });

  setContext<SettingsValidationContext>(SETTINGS_VALIDATION, {
    get issue() {
      return session.editor.validationIssue;
    },
    clear: () => {
      session.editor.validationIssue = null;
    },
  });

  async function revealValidationIssue() {
    const issue = session.editor.validationIssue;
    if (!issue || !visible || session.editor.connectionDraft) return;
    pendingConnectionAction = null;
    active = issue.section;
    await tick();
    // Voice validation currently identifies the workflow, rather than an individual option.
    if (issue.field === "voice-transcription") {
      for (const details of contentPane?.querySelectorAll("details") ?? [])
        details.open = true;
    }
    const control = issue.control
      ? contentPane?.querySelector<HTMLElement>(`#${CSS.escape(issue.control)}`)
      : null;
    for (
      let parent = control?.parentElement;
      parent;
      parent = parent.parentElement
    ) {
      if (parent instanceof HTMLDetailsElement) parent.open = true;
    }
    const target = control?.matches(
      "input, button, textarea, select, [tabindex]",
    )
      ? control
      : control?.querySelector<HTMLElement>(
          "input, button, textarea, select, [tabindex]",
        );
    const focus =
      target && !target.matches(":disabled, [aria-disabled=true]")
        ? target
        : contentPane?.querySelector<HTMLElement>("#settings-page-heading");
    focus?.focus({ preventScroll: true });
    focus?.scrollIntoView({ block: "nearest", behavior: "instant" });
  }

  let contentPane = $state<HTMLDivElement | null>(null);
  let headingHeight = $state(0);
  const contentKey = $derived(
    `${active}/${active === "connections" ? (session.editor.connectionDraft?.id ?? "list") : "section"}`,
  );
  $effect(() => {
    if (visible && contentKey)
      contentPane?.scrollTo({ top: 0, left: 0, behavior: "instant" });
  });

  let pendingConnectionAction = $state<(() => void) | null>(null);
  function withSavedSettings(action: () => void) {
    if (session.editor.saving) return;
    if (session.editor.runtimeDirty) pendingConnectionAction = action;
    else action();
  }
  function addConnection(purpose: Purpose) {
    withSavedSettings(() => {
      onOpenConnection({ id: "", purpose, create: true });
    });
  }
  async function continueConnection(save: boolean) {
    if (save) {
      if (!(await session.editor.save())) {
        await revealValidationIssue();
        return;
      }
      shortcutCapture.markSaved();
    }
    if (!save) session.editor.discardSettingsDraft();
    const action = pendingConnectionAction;
    pendingConnectionAction = null;
    action?.();
  }
  $effect(() => {
    if (!visible) pendingConnectionAction = null;
  });
  setContext(SETTINGS_NAVIGATION, selectSection);
  function browseConnections() {
    withSavedSettings(() => {
      onOpenConnection({
        id: "",
        purpose: workflowPurpose ?? Purpose.$zero,
        create: false,
      });
    });
  }
  function selectSection(id: SettingsSectionID) {
    if (onNavigate) {
      onNavigate(id);
      return;
    }
    if (id === "connections") {
      browseConnections();
      return;
    }
    active = id;
    if (id === "audio") void session.editor.refreshDevices();
  }

  async function saveSettings() {
    if (await session.editor.save()) {
      shortcutCapture.markSaved();
      onSaved();
    } else await revealValidationIssue();
  }

  // Settings has no transport to state its own progress, so the channel carries
  // both outcomes of whatever you just pressed.
  const messages = $derived.by(() => {
    const out: Message[] = [];
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

<div class="flex min-h-0 min-w-0 flex-1 overflow-hidden">
  <SettingsNav
  onOpenChain={onClose}
    {active}
    onSelect={selectSection}
    bind:navigationRef
    invalidSection={session.editor.validationIssue?.section}
  />

  <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div
      bind:this={contentPane}
      style:scroll-padding-top={`${headingHeight + 16}px`}
      class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 pb-6"
    >
      <section
        aria-labelledby="settings-section-title"
        class="@container mx-auto flex w-full max-w-[820px] flex-col gap-4"
      >
        <div
          bind:clientHeight={headingHeight}
          class="sticky top-0 z-10 space-y-1.5 border-b border-hairline bg-background pb-4 pt-5"
        >
          <h3
            id="settings-page-heading"
            tabindex="-1"
            class="font-display text-[15px] font-semibold tracking-tight"
          >
            {section.label}
          </h3>
          <p class="max-w-xl text-sm leading-5 text-muted-foreground">
            {section.blurb}
          </p>
        </div>
        <h2
          id="settings-section-title"
          class="sr-only"
          aria-live="polite"
          aria-atomic="true"
        >
          {section.label} settings
        </h2>
        {#if messages.length && pendingConnectionAction === null && !decisionOpen}<Notifications
            {messages}
          />{/if}
        {#if session.editor.validationIssue}
          <div
            class="flex flex-wrap items-center gap-2 border-l-2 border-destructive pl-3 text-sm"
          >
            <p
              id="settings-validation-message"
              role="alert"
              class="text-destructive"
            >
              {sectionByID(session.editor.validationIssue.section).label}: {session
                .editor.validationIssue.message}
            </p>
            {#if active !== session.editor.validationIssue.section}
              <Button variant="link" size="sm" onclick={revealValidationIssue}>
                Review {sectionByID(session.editor.validationIssue.section)
                  .label}
              </Button>
            {/if}
          </div>
        {/if}

        {#if session.editor.draft}
          {#if active === "voice-transcription" || active === "server" || active === "processing" || active === "speech"}
            <SavedConnectionPicker
              catalog={session.editor.draft.savedConnections}
              purpose={active === "voice-transcription"
                ? Purpose.Voice
                : active === "server"
                  ? Purpose.Transcription
                  : active === "processing"
                    ? Purpose.Cleanup
                    : Purpose.Speech}
              dirty={session.editor.dirty}
              busy={session.editor.saving ||
                session.editor.quickSettingsPending.length > 0}
              onChange={async (change) => {
                if (!session.editor.runtimeDirty)
                  return session.editor.changeConnection(change);
                withSavedSettings(() => {
                  void session.editor.changeConnection(change);
                });
                return false;
              }}
              onAdd={() =>
                addConnection(
                  active === "voice-transcription"
                    ? Purpose.Voice
                    : active === "server"
                      ? Purpose.Transcription
                      : active === "processing"
                        ? Purpose.Cleanup
                        : Purpose.Speech,
                )}
              onBrowse={browseConnections}
              onManage={() =>
                withSavedSettings(() => {
                  const purpose =
                    active === "voice-transcription"
                      ? Purpose.Voice
                      : active === "server"
                        ? Purpose.Transcription
                        : active === "processing"
                          ? Purpose.Cleanup
                          : Purpose.Speech;
                  const c =
                    session.editor.applied?.savedConnections.entries?.find(
                      (c) =>
                        c.id ===
                        session.editor.applied?.savedConnections.selected?.[
                          purpose
                        ],
                    );
                  onOpenConnection({
                    id: c?.id ?? "",
                    purpose,
                    create: false,
                  });
                })}
            />
          {/if}
          {#if active === "local-runtime"}
            <!-- The runtime inventory has a place of its own on the rail, with
                 room for the model table and the process output drawer. One
                 screen, one implementation: this points at it rather than
                 rendering a second, smaller copy here. -->
            <div class="rounded-lg border border-hairline p-4">
              <p class="text-[13px] text-secondary-foreground">
                Local runtimes have their own place, alongside the workflows on
                the rail. Install and remove engines, pick models, and read
                process output there.
              </p>
              <Button class="mt-3" size="sm" onclick={onOpenRuntimes}
                >Open Local runtime</Button
              >
            </div>
          {:else if active === "connections"}
            <Button onclick={browseConnections}>Open connections</Button>
          {:else if active === "voice-transcription"}
            {#if session.editor.draft.savedConnections.selected?.voice}
              <VoiceTranscriptionSettings
                runtime={session.runtime}
                onManageRuntime={() => selectSection("local-runtime")}
                editor={session.editor}
                settings={session.editor.draft}
                draft
                disabled={session.editor.saving}
                onAddConnection={addConnection}
              />
            {/if}
          {:else if active === "vocabulary"}
            <VocabularySection
              settings={session.editor.draft}
              onChange={(patch) =>
                Object.assign(session.editor.draft!.vocabulary, patch)}
              disabled={session.editor.saving}
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
              onChooseMicrophone={(choice) =>
                session.editor.chooseMicrophone(choice)}
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
                runtime={session.runtime}
                onManageRuntime={() => selectSection("local-runtime")}
                onEnter={() =>
                  void session.editor.ensureConnectionMetadata(
                    Purpose.Transcription,
                    true,
                  )}
                metadataStatus={session.editor.connectionMetadataStatus(
                  Purpose.Transcription,
                )}
                connectionStale={session.editor.connectionResultStale(
                  Purpose.Transcription,
                )}
                draftModels={session.editor.modelDraftIDs(
                  Purpose.Transcription,
                )}
                onChooseModel={(model) =>
                  session.editor.chooseModel(Purpose.Transcription, model)}
                onForgetModel={() =>
                  session.editor.forgetModel(Purpose.Transcription)}
                bind:settings={session.editor.draft}
                connection={session.editor.connection}
                busy={session.editor.sttConnectionTesting}
                onTestConnection={() => session.editor.testConnection()}
              />
            {/if}
          {:else if active === "processing"}
            {#if session.editor.draft.savedConnections.selected?.cleanup}
              <ProcessingSection
                runtime={session.runtime}
                onManageRuntime={() => selectSection("local-runtime")}
                onEnter={() =>
                  void session.editor.ensureConnectionMetadata(
                    Purpose.Cleanup,
                    true,
                  )}
                metadataStatus={session.editor.connectionMetadataStatus(
                  Purpose.Cleanup,
                )}
                connectionStale={session.editor.connectionResultStale(
                  Purpose.Cleanup,
                )}
                draftModels={session.editor.modelDraftIDs(Purpose.Cleanup)}
                onChooseModel={(model) =>
                  session.editor.chooseModel(Purpose.Cleanup, model)}
                onForgetModel={() =>
                  session.editor.forgetModel(Purpose.Cleanup)}
                bind:settings={session.editor.draft}
                profiles={session.editor.processingProfiles}
                connection={session.editor.processingConnection}
                busy={session.editor.processingConnectionTesting}
                onTestConnection={() =>
                  session.editor.testPostProcessingConnection()}
              />
            {/if}
          {:else if active === "speech"}
            {#if session.editor.draft.savedConnections.selected?.speech}
              <SpeechSection
                runtime={session.runtime}
                onManageRuntime={() => selectSection("local-runtime")}
                onEnter={() =>
                  void session.editor.ensureConnectionMetadata(
                    Purpose.Speech,
                    true,
                  )}
                metadataStatus={session.editor.connectionMetadataStatus(
                  Purpose.Speech,
                )}
                voices={session.editor.voices}
                voicesBusy={session.editor.voicesBusy}
                onDiscoverVoices={() => session.editor.discoverVoices()}
                connectionStale={session.editor.connectionResultStale(
                  Purpose.Speech,
                )}
                draftModels={session.editor.modelDraftIDs(Purpose.Speech)}
                onChooseModel={(model) =>
                  session.editor.chooseModel(Purpose.Speech, model)}
                onForgetModel={() => session.editor.forgetModel(Purpose.Speech)}
                bind:settings={session.editor.draft}
                status={session.speech.status}
                busy={session.speech.previewing}
                connection={session.editor.ttsConnection}
                connectionBusy={session.editor.ttsConnectionTesting}
                canPreview={session.dictation.status.state === State.Idle &&
                  !session.files.status.canCancel}
                onPreview={() => {
                  if (session.editor.draft)
                    void session.speech.previewVoice(session.editor.draft);
                }}
                onStop={() => session.speech.stopTTS()}
                onSave={() => session.speech.saveTTSAudio()}
                onClear={() => session.speech.clearTTSAudio()}
                onTestConnection={() =>
                  session.editor.testTextToSpeechConnection()}
              />
            {/if}
          {:else if active === "history"}
            <HistorySection
              bind:settings={session.editor.draft}
              enabled={session.editor.applied?.historyEnabled ?? false}
              entries={session.history.entries}
              onCopy={(id) => session.history.copyHistoryEntry(id)}
              onCopyVersion={(id, version) =>
                session.history.copyHistoryEntryVersion(id, version)}
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
      class="flex min-h-14 shrink-0 flex-wrap items-center justify-end gap-2.5 border-t border-hairline px-5 py-2.5"
    >
      {#if session.editor.draft}
        <span
          class="figure mr-auto flex items-center gap-2 text-xs font-medium text-muted-foreground"
          aria-live="polite"
          aria-atomic="true"
        >
          <StatusBadge
            tone={dirty || session.editor.saving ? "accent" : "success"}
            dot
          >
            {session.editor.saving
              ? "Saving changes…"
              : dirty
                ? "Unsaved changes"
                : "All changes saved"}
          </StatusBadge>
        </span>
        {#if session.editor.runtimeDirty && active !== "connections"}
          <Button
            variant="ghost"
            disabled={session.editor.saving}
            onclick={() => session.editor.discardSettingsDraft()}
            >Discard changes</Button
          >
        {/if}
        <Button
          variant="outline"
          disabled={session.editor.saving}
          onclick={onClose}>Done</Button
        >
        {#if active !== "connections"}<Button
            disabled={session.busy || shortcutCapture.capturing || !dirty}
            onclick={saveSettings}
          >
            {#if session.editor.saving}
              <LoaderCircleIcon data-icon="inline-start" class="animate-spin" />
            {/if}
            {session.editor.saving
              ? "Saving…"
              : saveReturnsToTask
                ? "Save and return"
                : "Save"}
          </Button>{/if}
      {:else}
        <Button
          variant="outline"
          disabled={session.editor.saving}
          onclick={onClose}>Done</Button
        >
      {/if}
    </div>
  </div>
</div>

<PendingChangesDialog
  open={pendingConnectionAction !== null}
  busy={session.editor.saving}
  title="Save settings before continuing?"
  description="Your model and task edits have not been applied. Save them for the current connection, or discard them before continuing."
  error={session.messages.error}
  discardLabel="Discard and continue"
  discardVariant="secondary"
  saveLabel="Save and continue"
  onKeepEditing={() => (pendingConnectionAction = null)}
  onDiscard={() => continueConnection(false)}
  onSave={() => continueConnection(true)}
/>
