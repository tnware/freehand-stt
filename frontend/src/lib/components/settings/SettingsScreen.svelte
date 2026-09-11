<script lang="ts">
  import { SETTINGS_NAVIGATION } from "$lib/navigation";
  import VocabularySection from "./sections/VocabularySection.svelte";
  import { setContext, tick, untrack } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import * as Dialog from "$lib/components/ui/dialog";
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
  import { State } from "$lib/state";
  import type { Message } from "$lib/utils/messages";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import { cn } from "$lib/utils";

  let {
    session,
    visible = true,
    active = $bindable(),
    navigationRef = $bindable(null),
    onClose,
    overlayPreviewing,
    onStartOverlayPreview,
    onStopOverlayPreview,
  }: {
    session: Session;
    visible?: boolean;
    active: SettingsSectionID;
    navigationRef?: HTMLElement | null;
    onClose: () => void;
    overlayPreviewing: boolean;
    onStartOverlayPreview: () => void;
    onStopOverlayPreview: () => void;
  } = $props();

  const section = $derived(sectionByID(active));
  const dirty = $derived(session.editor.dirty);
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
    const id = purpose && session.editor.applied?.savedConnections.selected?.[purpose];
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
      for (const details of contentPane?.querySelectorAll("details") ?? []) details.open = true;
    }
    const control = issue.control
      ? contentPane?.querySelector<HTMLElement>(`#${CSS.escape(issue.control)}`)
      : null;
    for (let parent = control?.parentElement; parent; parent = parent.parentElement) {
      if (parent instanceof HTMLDetailsElement) parent.open = true;
    }
    const target = control?.matches("input, button, textarea, select, [tabindex]")
      ? control
      : control?.querySelector<HTMLElement>("input, button, textarea, select, [tabindex]");
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
    if (visible && contentKey) contentPane?.scrollTo({ top: 0, left: 0, behavior: "instant" });
  });

  let pendingConnectionAction = $state<(() => void) | null>(null);
  function preventDismissWhileSaving(event: Event) {
    // onOpenChange observes a close; these hooks can prevent it.
    if (session.editor.saving) event.preventDefault();
  }
  function withSavedSettings(action: () => void) {
    if (session.editor.saving) return;
    if (session.editor.runtimeDirty) pendingConnectionAction = action;
    else action();
  }
  function addConnection(purpose: Purpose) {
    withSavedSettings(() => {
      void WindowingService.OpenConnectionManager({ id: "", purpose, create: true }).catch(
        (cause) => session.messages.reportFailure(String(cause)),
      );
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
      void WindowingService.OpenConnectionManager({
        id: "",
        purpose: Purpose.$zero,
        create: false,
      }).catch((cause) => session.messages.reportFailure(String(cause)));
    });
  }
  function selectSection(id: SettingsSectionID) {
    if (id === "connections") {
      browseConnections();
      return;
    }
    active = id;
    if (id === "audio") void session.editor.refreshDevices();
  }

  async function saveSettings() {
    if (await session.editor.save()) shortcutCapture.markSaved();
    else await revealValidationIssue();
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
  <SettingsNav
    {active}
    onSelect={selectSection}
    bind:navigationRef
    invalidSection={session.editor.validationIssue?.section}
  />

  <div class="flex min-w-0 flex-1 flex-col">
    <div
      bind:this={contentPane}
      style:scroll-padding-top={`${headingHeight + 16}px`}
      class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-6 sm:px-7"
    >
      <section
        aria-labelledby="settings-section-title"
        class="@container mx-auto flex w-full max-w-[820px] flex-col gap-5"
      >
        <div
          bind:clientHeight={headingHeight}
          class="sticky top-0 z-10 space-y-2 bg-background py-6"
        >
          <h3
            id="settings-page-heading"
            tabindex="-1"
            class="font-display text-[26px] font-medium tracking-tight"
          >
            {section.label}
          </h3>
          <p class="max-w-xl text-[13px] leading-relaxed text-muted-foreground">
            {section.blurb}
          </p>
        </div>
        <h2 id="settings-section-title" class="sr-only" aria-live="polite" aria-atomic="true">
          {section.label} settings
        </h2>
        {#if messages.length && pendingConnectionAction === null}<Notifications {messages} />{/if}
        {#if session.editor.validationIssue}
          <div class="flex flex-wrap items-center gap-2 border-l-2 border-destructive pl-3 text-sm">
            <p id="settings-validation-message" role="alert" class="text-destructive">
              {sectionByID(session.editor.validationIssue.section).label}: {session.editor
                .validationIssue.message}
            </p>
            {#if active !== session.editor.validationIssue.section}
              <Button variant="link" size="sm" onclick={revealValidationIssue}>
                Review {sectionByID(session.editor.validationIssue.section).label}
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
              busy={session.editor.saving || session.editor.quickSettingsPending.length > 0}
              onChange={async (change) => {
                if (!session.editor.runtimeDirty) return session.editor.changeConnection(change);
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
                  const c = session.editor.applied?.savedConnections.entries?.find(
                    (c) => c.id === session.editor.applied?.savedConnections.selected?.[purpose],
                  );
                  void WindowingService.OpenConnectionManager({
                    id: c?.id ?? "",
                    purpose,
                    create: false,
                  }).catch((cause) => session.messages.reportFailure(String(cause)));
                })}
            />
          {/if}
          {#if active === "connections"}
            <Button onclick={browseConnections}>Open connections</Button>
          {:else if active === "voice-transcription"}
            {#if session.editor.draft.savedConnections.selected?.voice}<div
                class="border-b border-hairline pb-4"
              >
                <VoiceTranscriptionSettings
                  editor={session.editor}
                  settings={session.editor.draft}
                  draft
                  disabled={session.editor.saving}
                  onAddConnection={addConnection}
                />
              </div>{/if}
          {:else if active === "vocabulary"}
            <VocabularySection
              settings={session.editor.draft}
              onChange={(patch) => Object.assign(session.editor.draft!.vocabulary, patch)}
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
                connectionStale={session.editor.connectionResultStale(Purpose.Transcription)}
                draftModels={session.editor.modelDraftIDs(Purpose.Transcription)}
                onChooseModel={(model) => session.editor.chooseModel(Purpose.Transcription, model)}
                onForgetModel={() => session.editor.forgetModel(Purpose.Transcription)}
                bind:settings={session.editor.draft}
                connection={session.editor.connection}
                busy={session.editor.sttConnectionTesting}
                onTestConnection={() => session.editor.testConnection()}
              />
            {/if}
          {:else if active === "processing"}
            {#if session.editor.draft.savedConnections.selected?.cleanup}
              <ProcessingSection
                connectionStale={session.editor.connectionResultStale(Purpose.Cleanup)}
                draftModels={session.editor.modelDraftIDs(Purpose.Cleanup)}
                onChooseModel={(model) => session.editor.chooseModel(Purpose.Cleanup, model)}
                onForgetModel={() => session.editor.forgetModel(Purpose.Cleanup)}
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
                voices={session.editor.voices}
                voicesBusy={session.editor.voicesBusy}
                onDiscoverVoices={() => session.editor.discoverVoices()}
                connectionStale={session.editor.connectionResultStale(Purpose.Speech)}
                draftModels={session.editor.modelDraftIDs(Purpose.Speech)}
                onChooseModel={(model) => session.editor.chooseModel(Purpose.Speech, model)}
                onForgetModel={() => session.editor.forgetModel(Purpose.Speech)}
                bind:settings={session.editor.draft}
                status={session.speech.status}
                busy={session.speech.previewing}
                connection={session.editor.ttsConnection}
                connectionBusy={session.editor.ttsConnectionTesting}
                canPreview={session.dictation.status.state === State.Idle &&
                  !session.files.status.canCancel}
                onPreview={() => {
                  if (session.editor.draft) void session.speech.previewVoice(session.editor.draft);
                }}
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
      class="flex min-h-[64px] shrink-0 flex-wrap items-center justify-end gap-2.5 border-t border-hairline bg-background px-6 py-3"
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
        {#if session.editor.runtimeDirty && active !== "connections"}
          <Button
            variant="ghost"
            disabled={session.editor.saving}
            onclick={() => session.editor.discardSettingsDraft()}>Discard changes</Button
          >
        {/if}
        <Button variant="outline" disabled={session.editor.saving} onclick={onClose}>Close</Button>
        {#if active !== "connections"}<Button
            disabled={session.busy || shortcutCapture.capturing || !dirty}
            onclick={saveSettings}
          >
            {#if session.editor.saving}
              <LoaderCircleIcon data-icon="inline-start" class="animate-spin" />
            {/if}
            {session.editor.saving ? "Saving…" : "Save settings"}
          </Button>{/if}
      {:else}
        <Button variant="outline" disabled={session.editor.saving} onclick={onClose}>Close</Button>
      {/if}
    </div>
  </div>
</div>

<Dialog.Root
  open={pendingConnectionAction !== null}
  onOpenChange={(open) => {
    if (!open && !session.editor.saving) pendingConnectionAction = null;
  }}
>
  <Dialog.Content
    showCloseButton={!session.editor.saving}
    onEscapeKeydown={preventDismissWhileSaving}
    onInteractOutside={preventDismissWhileSaving}
  >
    <Dialog.Header>
      <Dialog.Title>Save settings before changing connections?</Dialog.Title>
      <Dialog.Description
        >Your model and task edits have not been applied. Save them for the current connection, or
        discard them before continuing.</Dialog.Description
      >
    </Dialog.Header>
    {#if session.messages.error}<p role="alert" class="text-destructive">
        {session.messages.error}
      </p>{/if}
    <Dialog.Footer>
      <Button
        variant="outline"
        disabled={session.editor.saving}
        onclick={() => (pendingConnectionAction = null)}>Keep editing</Button
      >
      <Button
        variant="secondary"
        disabled={session.editor.saving}
        onclick={() => continueConnection(false)}>Discard and continue</Button
      >
      <Button disabled={session.editor.saving} onclick={() => continueConnection(true)}
        >Save and continue</Button
      >
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
