<script lang="ts">
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import SettingsIcon from "@lucide/svelte/icons/settings-2";
  import MicIcon from "@lucide/svelte/icons/mic";
  import AudioLinesIcon from "@lucide/svelte/icons/audio-lines";
  import TextIcon from "@lucide/svelte/icons/text";
  import ArrowUpRightIcon from "@lucide/svelte/icons/arrow-up-right";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import { microphoneChoiceFor, microphoneLabel } from "$lib/utils/microphone";
  import { Switch } from "$lib/components/ui/switch";
  import { Button } from "$lib/components/ui/button";
  import * as Popover from "$lib/components/ui/popover";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import QuickSaveStatus from "$lib/components/settings/QuickSaveStatus.svelte";
  import ManagedRuntimeControls from "./ManagedRuntimeControls.svelte";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import type { Session } from "$lib/stores/session.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import { Role } from "$bindings/compatibility";
  import { FileTranscriptionPhase, State, TTSPhase } from "$lib/state";

  let {
    session,
    workflow,
    title,
    onAddConnection,
    onOpenOptions,
    onOpenMicrophone,
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
    onOpenMicrophone?: () => void;
    onOpenCleanup: () => void;
    onOpenVocabulary: () => void;
    onOpenDelivery: () => void;
    onOpenOverlay: () => void;
    onOpenRuntime: () => void;
    disabled?: boolean;
    editing?: boolean;
  } = $props();
  const uid = $props.id();
  let modelOpen = $state(false);
  let preferencesOpen = $state(false);
  const settings = $derived(session.editor.applied ?? session.editor.draft);
  const purpose = $derived(
    workflow === "voice"
      ? Purpose.Voice
      : workflow === "tts"
        ? Purpose.Speech
        : Purpose.Transcription,
  );
  const config = $derived(
    workflow === "voice"
      ? settings?.voiceTranscription
      : workflow === "tts"
        ? settings?.textToSpeech
        : settings,
  );
  const connection = $derived(
    settings?.savedConnections.entries?.find(
      (entry) => entry.id === settings.savedConnections.selected?.[purpose],
    ),
  );
  const instance = $derived(
    session.runtime.statusFor(config?.managedInstanceID ?? ""),
  );
  const profiles = $derived(
    workflow === "tts"
      ? settings?.compatibilityProfiles.speech
      : settings?.compatibilityProfiles.transcription,
  );
  const backend = $derived(
    profiles?.find((profile) => profile.id === config?.compatibilityProfile),
  );
  const managed = $derived(!!config?.managedInstanceID);
  const modelID = $derived(
    managed
      ? ((workflow === "tts"
          ? instance?.instance.speechModel
          : instance?.instance.model) ?? "")
      : (config?.model ?? ""),
  );
  const modelName = $derived(
    managed
      ? instance?.status.models?.find((model) => model.id === modelID)?.name ||
          modelID
      : modelID,
  );
  const serverLoaded = $derived(
    !!backend?.capabilities.serverLoadedModel &&
      !(workflow === "voice" && backend?.capabilities.realtime),
  );
  const engine = $derived(
    managed
      ? session.runtime.providers.find(
          (provider) => provider.id === instance?.instance.provider,
        )?.name || "Local runtime"
      : backend?.label || connection?.name || "No connection selected",
  );
  const metadata = $derived(
    workflow === "voice"
      ? session.editor.currentVoiceConnection
      : workflow === "tts"
        ? session.editor.ttsConnection
        : session.editor.connection,
  );
  const testing = $derived(
    workflow === "voice"
      ? session.editor.voiceConnectionTesting
      : workflow === "tts"
        ? session.editor.ttsConnectionTesting
        : session.editor.sttConnectionTesting,
  );
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
  const controlsDisabled = $derived(
    disabled ||
      editing ||
      session.editor.dirty ||
      session.editor.saving ||
      !session.editor.applied ||
      session.editor.quickSettingsPending.length > 0 ||
      !!settings?.configuration.recoveryRequired,
  );
  const field = $derived(
    workflow === "voice"
      ? "voice-transcription"
      : workflow === "tts"
        ? "speech-controls"
        : "stt-model",
  );
  function chooseModel(model: string) {
    if (controlsDisabled || managed) return Promise.resolve(false);
    return session.editor.updateQuickSettings(
      workflow === "voice"
        ? { voiceTranscription: { model } }
        : workflow === "tts"
          ? { textToSpeech: { model } }
          : { model },
      field,
    );
  }
  function navigate(action: () => void) {
    modelOpen = false;
    preferencesOpen = false;
    action();
  }
  $effect(() => {
    workflow;
    modelOpen = false;
    preferencesOpen = false;
  });
</script>

<aside
  class="workflow-settings flex w-[252px] shrink-0 flex-col border-r border-hairline bg-layer-fill"
  aria-label={`${title} settings`}
>
  <SidebarHeader {title} />
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto px-3 py-5">
    {#if settings && config}
      <div
        class="space-y-2"
        role="group"
        aria-label={`${title} quick settings`}
      >
        <p class="text-xs font-medium text-muted-foreground">Model</p>
        <Popover.Root bind:open={modelOpen}>
          <Popover.Trigger
            class="model-choice"
            disabled={controlsDisabled}
            aria-label="Change engine or model"
          >
            <span class="model-identity">
              <ProviderIcon
                profile={managed
                  ? instance?.instance.provider
                  : config.compatibilityProfile}
                size={36}
              />
              <span class="min-w-0 flex-1">
                <span
                  class="block break-words text-[13px] font-medium [overflow-wrap:anywhere]"
                  >{engine}</span
                >
                <span class="mt-1 block text-[11px] text-muted-foreground"
                  >{managed ? "Local runtime" : "Server connection"}</span
                >
              </span>
            </span>
            <span class="model-selection">
              <span
                class="min-w-0 flex-1 break-words text-sm font-semibold [overflow-wrap:anywhere]"
                >{connection
                  ? !managed && serverLoaded
                    ? "Server-loaded model"
                    : modelName || "Choose a model"
                  : "Choose a model"}</span
              >
              <ChevronDownIcon class="size-4 shrink-0 text-muted-foreground" />
            </span>
          </Popover.Trigger>
          <Popover.Content
            align="start"
            class="w-[360px] max-w-[calc(100vw-24px)] space-y-4"
            role="dialog"
            aria-label="Engine and model"
          >
            <div>
              <h3 class="text-sm font-semibold">Engine and model</h3>
              <p class="mt-1 text-xs text-muted-foreground">
                Use a saved connection or a local runtime.
              </p>
            </div>
            <div class="space-y-1.5">
              <label for={`${uid}-connection`} class="content-value"
                >Connection</label
              >
              <ConnectionSelect
                id={`${uid}-connection`}
                catalog={settings.savedConnections}
                runtimeInstances={session.runtime.instances}
                {purpose}
                disabled={controlsDisabled || testing}
                onChange={(change) =>
                  controlsDisabled
                    ? Promise.resolve(false)
                    : session.editor.changeConnection(change)}
                onAdd={() => navigate(() => onAddConnection(purpose))}
              />
            </div>
            {#if managed}
              <ManagedRuntimeControls
                runtime={session.runtime}
                instanceID={config.managedInstanceID ?? ""}
                workBusy={runtimeWorkBusy}
                disabled={controlsDisabled}
                role={workflow === "tts" ? Role.Speech : Role.Transcription}
                onManage={() => navigate(onOpenRuntime)}
              />
            {:else}
              <RuntimeModelPicker
                id={`${uid}-workflow-model`}
                value={config.model}
                compact
                immediate
                showProfileName={false}
                models={session.editor.connectionResultStale(purpose, settings)
                  ? []
                  : (metadata?.modelIDs ?? [])}
                savedModels={rememberedModels(settings, purpose).map(
                  (entry) => entry.model,
                )}
                {serverLoaded}
                disabled={controlsDisabled || !connection}
                busy={testing}
                onChoose={chooseModel}
                onDiscover={() =>
                  void session.editor.testAppliedConnection(purpose)}
                onEnter={() =>
                  void session.editor.ensureConnectionMetadata(purpose, true)}
                metadataStatus={session.editor.connectionMetadataStatus(
                  purpose,
                )}
              />
            {/if}
            <QuickSaveStatus
              fields={[field]}
              pending={session.editor.quickSettingsPending}
              saved={session.editor.quickSettingsSaved}
              failed={session.editor.quickSettingsFailed}
            />
            <div class="flex flex-wrap gap-2 border-t border-hairline pt-3">
              <Button
                variant="outline"
                size="sm"
                onclick={() => navigate(onOpenRuntime)}
                >Browse local models</Button
              ><Button
                variant="ghost"
                size="sm"
                onclick={() => navigate(onOpenOptions)}>Model options</Button
              >
            </div>
          </Popover.Content>
        </Popover.Root>
        {#if managed && instance && instance.status.state !== "running"}<p
            class="text-xs text-muted-foreground"
          >
            Local runtime is {instance.status.state}. Open the model chooser to
            manage it.
          </p>{/if}
      </div>
      <section class="workflow-context" aria-label="Workflow summary">
        <h2 class="context-heading">Workflow</h2>
        {#if workflow === "voice"}
          <button
            class="context-row"
            {disabled}
            onclick={() => navigate(onOpenMicrophone ?? onOpenOptions)}
            aria-label="Microphone settings"
          >
            <MicIcon class="size-4 shrink-0 text-muted-foreground" />
            <span class="min-w-0 flex-1"
              ><span class="context-label">Microphone</span><span
                class="context-value"
                >{microphoneLabel(
                  microphoneChoiceFor(settings.microphoneID),
                  session.editor.devices,
                )}</span
              ></span
            >
          </button>
        {:else if workflow === "tts"}
          <button
            class="context-row"
            {disabled}
            onclick={() => navigate(onOpenOptions)}
            aria-label="Speech voice settings"
          >
            <AudioLinesIcon class="size-4 shrink-0 text-muted-foreground" />
            <span class="min-w-0 flex-1"
              ><span class="context-label">Voice</span><span
                class="context-value"
                >{settings.textToSpeech.voice || "Choose a voice"}</span
              ></span
            >
          </button>
        {/if}
        {#if workflow !== "tts"}
          <button
            class="context-row"
            {disabled}
            onclick={() => navigate(onOpenCleanup)}
            aria-label="Transcript cleanup settings"
          >
            <TextIcon class="size-4 shrink-0 text-muted-foreground" />
            <span class="min-w-0 flex-1"
              ><span class="context-label">Transcript cleanup</span><span
                class="context-value"
                >{settings.postProcessing.enabled
                  ? settings.savedConnections.selected?.cleanup
                    ? "Enabled"
                    : "Connection required"
                  : "Off"}</span
              ></span
            >
          </button>
        {/if}
        <div class="context-row">
          <ArrowUpRightIcon class="size-4 shrink-0 text-muted-foreground" />
          <span class="min-w-0 flex-1"
            ><span class="context-label">Output</span><span
              class="context-value"
              >{workflow === "voice"
                ? settings.autoInsert
                  ? "Original app · copy if blocked"
                  : "Manual copy"
                : workflow === "file"
                  ? "Manual copy"
                  : "Play or save audio"}</span
            ></span
          >
        </div>
      </section>
      {#if workflow === "voice" && settings.toggleShortcut}
        <div class="shortcut-summary">
          <span class="context-label">Recording shortcut</span><ShortcutKeys
            value={settings.toggleShortcut}
            label="Toggle recording shortcut"
          />
        </div>
      {/if}
    {/if}

    <div class="mt-auto pt-8">
      <Popover.Root bind:open={preferencesOpen}>
        <Popover.Trigger
          class="preferences-button"
          disabled={disabled || session.editor.saving}
          aria-label="Workflow preferences"
          ><SettingsIcon class="size-4" />Preferences</Popover.Trigger
        >
        <Popover.Content
          align="start"
          class="w-[300px] max-w-[calc(100vw-24px)] space-y-1"
          role="dialog"
          aria-label="Workflow preferences"
        >
          <h3 class="px-2 pb-2 text-sm font-semibold">Preferences</h3>
          <Button
            variant="ghost"
            class="w-full justify-start"
            onclick={() => navigate(onOpenOptions)}
            >{workflow === "voice"
              ? "Voice and microphone settings"
              : workflow === "file"
                ? "Transcription settings"
                : "Voice and speech settings"}</Button
          >
          {#if workflow !== "tts"}
            <Button
              variant="ghost"
              class="w-full justify-start"
              onclick={() => navigate(onOpenCleanup)}>Transcript cleanup</Button
            >
            <Button
              variant="ghost"
              class="w-full justify-start"
              onclick={() => navigate(onOpenVocabulary)}
              >Personal vocabulary</Button
            >
          {/if}
          {#if workflow === "voice"}
            <Button
              variant="ghost"
              class="w-full justify-start"
              onclick={() => navigate(onOpenDelivery)}>Text delivery</Button
            >
            <Button
              variant="ghost"
              class="w-full justify-start"
              onclick={() => navigate(onOpenOverlay)}>Recording overlay</Button
            >
          {/if}
          {#if workflow === "file"}<div
              class="flex items-center justify-between gap-3 border-t border-hairline px-2 py-3"
            >
              <span class="text-sm">Show results as they arrive</span><Switch
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
              />
            </div>{/if}
        </Popover.Content>
      </Popover.Root>
    </div>
  </div>
</aside>

<style>
  :global(.workflow-settings .model-choice) {
    display: flex;
    flex-direction: column;
    gap: 0;
    width: 100%;
    padding: 0;
    text-align: left;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    background: var(--well);
    transition: background-color 120ms;
  }
  .model-identity {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.875rem 0.75rem;
    width: 100%;
    border-bottom: 1px solid var(--hairline);
    background: var(--subtle-fill);
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  }
  .model-selection {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem;
  }
  .workflow-context {
    margin-top: 1.75rem;
  }
  .context-heading {
    margin-bottom: 0.5rem;
    font-size: 11px;
    font-weight: 600;
    color: var(--muted-foreground);
  }
  .context-row {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    width: 100%;
    padding: 0.625rem 0.375rem;
    border-radius: var(--radius-md);
    text-align: left;
  }
  button.context-row:hover:not(:disabled) {
    background: var(--subtle-fill-hover);
  }
  button.context-row:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
  .context-label {
    display: block;
    font-size: 11px;
    color: var(--muted-foreground);
  }
  .context-value {
    display: block;
    margin-top: 0.125rem;
    font-size: 12px;
    color: var(--secondary-foreground);
    overflow-wrap: anywhere;
  }
  .shortcut-summary {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--hairline);
  }
  :global(.workflow-settings .model-choice:hover:not(:disabled)) {
    background: var(--subtle-fill-hover);
  }
  :global(.workflow-settings .model-choice:focus-visible),
  :global(.workflow-settings .preferences-button:focus-visible) {
    outline: 2px solid var(--ring);
    outline-offset: 2px;
  }
  :global(.workflow-settings .model-choice:disabled) {
    opacity: 0.65;
  }
  :global(.workflow-settings .preferences-button) {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--secondary-foreground);
    font-size: 13px;
    padding: 0.375rem 0;
  }
  :global(.workflow-settings .preferences-button:hover) {
    color: var(--foreground);
  }
</style>
