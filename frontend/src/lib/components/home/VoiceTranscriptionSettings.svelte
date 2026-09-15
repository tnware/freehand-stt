<script lang="ts">
  import SettingsCard from "../settings/SettingsCard.svelte";
  import SettingsDisclosure from "../settings/SettingsDisclosure.svelte";
  import RuntimeModelPicker from "../settings/RuntimeModelPicker.svelte";
  import QuickSaveStatus from "../settings/QuickSaveStatus.svelte";
  import VocabularyLink from "$lib/components/settings/VocabularyLink.svelte";
  import ModelProfilePicker from "../settings/ModelProfilePicker.svelte";
  import { ID } from "$bindings/modelprofile";
  import LanguagePicker from "$lib/components/settings/LanguagePicker.svelte";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import { Purpose } from "$bindings/savedconnection";
  import type { Settings } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Input } from "$lib/components/ui/input";
  import { Textarea } from "$lib/components/ui/textarea";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import ManagedRuntimeControls from "./ManagedRuntimeControls.svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";

  let {
    editor,
    settings: suppliedSettings,
    disabled,
    draft: draftMode = false,
    setup: setupMode = false,
    sidebar = false,
    onOpenAdvanced,
    onAddConnection,
    runtime,
    runtimeWorkBusy = false,
    onManageRuntime = () => {},
  }: {
    editor: SettingsEditor;
    settings: Settings;
    disabled: boolean;
    draft?: boolean;
    setup?: boolean;
    /** Flat applied controls for the workflow sidebar; advanced options remain in the inspector. */
    sidebar?: boolean;
    onOpenAdvanced?: () => void;
    onAddConnection: (purpose: Purpose) => void;
    runtime?: ManagedRuntimeState;
    runtimeWorkBusy?: boolean;
    onManageRuntime?: () => void;
  } = $props();
  const uid = $props.id();
  const draft = $derived(draftMode && !sidebar);
  const setup = $derived(setupMode && !sidebar);
  const settings = $derived(
    sidebar ? (editor.applied ?? suppliedSettings) : suppliedSettings,
  );
  // The workspace remains mounted while Settings is open. Draft controls keep
  // their validation IDs; quick/setup controls need their own label targets.
  const controlID = (id: string) => (draft ? id : `${uid}-${id}`);
  const cfg = $derived(settings.voiceTranscription);
  const instanceID = $derived(cfg.managedInstanceID ?? "");
  const managed = $derived(!!instanceID);
  const row = $derived(runtime?.statusFor(instanceID));
  const localModel = $derived(
    row?.status.models?.find((model) => model.id === row.instance.model),
  );
  let profileNotice = $state("");
  const backend = $derived(
    settings.compatibilityProfiles.transcription?.find(
      (p) => p.id === cfg.compatibilityProfile,
    ),
  );
  const serverLoaded = $derived(
    backend?.capabilities.serverLoadedModel && !backend?.capabilities.realtime,
  );
  const profile = $derived(
    managed
      ? (localModel?.behavior ??
          settings.modelProfiles.voiceTranscription?.find(
            (p) => p.id === cfg.modelProfile,
          ))
      : settings.modelProfiles.voiceTranscription?.find(
          (p) => p.id === cfg.modelProfile,
        ),
  );
  function chooseProfile(value: string) {
    const selected = settings.modelProfiles.voiceTranscription?.find(
      (p) => p.id === value,
    );
    if (!selected || selected.id === cfg.modelProfile) return;
    const language =
      selected.languages?.length &&
      !selected.languages.some((l) => l.code === cfg.language)
        ? (selected.languages[0]?.code ?? "auto")
        : cfg.language;
    profileNotice =
      language !== cfg.language
        ? `The previous language is unavailable. ${selected.languages?.find((l) => l.code === language)?.label ?? language} selected.`
        : "";
    update({
      language,
      modelProfile: selected.id,
      realtime: cfg.realtime && selected.capabilities.realtime,
      transcriptionOptions: {
        prompt: "",
        temperatureOverride: false,
        temperature: 0,
      },
    });
  }
  const connectionID = $derived(
    settings.savedConnections.selected?.[Purpose.Voice] ?? "",
  );
  const busy = $derived(
    disabled ||
      (sidebar &&
        (!editor.applied ||
          editor.dirty ||
          editor.quickSettingsPending.length > 0 ||
          settings.configuration.recoveryRequired)) ||
      (managed && runtime?.isBusy(instanceID)) ||
      editor.saving ||
      editor.isQuickSettingsPending("voice-transcription"),
  );
  const testing = $derived(editor.voiceConnectionTesting);
  const availableModels = $derived(
    editor.currentVoiceConnection?.modelIDs ?? [],
  );
  const realtimeAvailable = $derived(
    !!profile?.capabilities.realtime && !!backend?.capabilities.realtime,
  );
  function chooseModel(model: string) {
    if (busy || managed) return false;
    return draft
      ? editor.chooseModel(Purpose.Voice, model)
      : editor.updateQuickSettings(
          { voiceTranscription: { model } },
          "voice-transcription",
        );
  }
  function update(patch: Partial<Settings["voiceTranscription"]>) {
    if (busy) return;
    if (draft) {
      if (patch.model !== undefined)
        editor.chooseModel(Purpose.Voice, patch.model);
      Object.assign(settings.voiceTranscription, patch);
    } else
      void editor.updateQuickSettings(
        { voiceTranscription: patch },
        "voice-transcription",
      );
  }
  function test() {
    void editor.testAppliedConnection(Purpose.Voice);
  }
  function setRealtime(realtime: boolean) {
    if (busy || (sidebar && realtime && !realtimeAvailable)) return;
    update({ realtime });
  }
</script>

<div
  class={draft
    ? "flex flex-col gap-3"
    : sidebar
      ? "flex min-w-0 flex-col gap-2"
      : "space-y-4"}
  class:voice-sidebar={sidebar}
>
  {#if !draft}
    {#if !setup && !sidebar}<h3 class="text-[13px] font-semibold">
        Transcription
      </h3>{/if}
    <div class="space-y-1.5">
      <label
        for={controlID("voice-connection")}
        class="text-[13px] font-semibold">Connection</label
      >
      <div class="flex gap-2">
        <ConnectionSelect
          id={controlID("voice-connection")}
          catalog={settings.savedConnections}
          runtimeInstances={runtime?.instances}
          purpose={Purpose.Voice}
          compact={sidebar}
          disabled={busy || testing}
          onChange={(change) =>
            busy || testing
              ? Promise.resolve(false)
              : editor.changeConnection(change)}
          onAdd={() => onAddConnection(Purpose.Voice)}
        />
      </div>
    </div>
    {#if managed && runtime}
      <ManagedRuntimeControls
        {sidebar}
        workBusy={runtimeWorkBusy}
        {instanceID}
        {runtime}
        disabled={disabled ||
          editor.saving ||
          editor.quickSettingsPending.length > 0}
        onManage={onManageRuntime}
      />
    {/if}
  {/if}
  {#if draft}
    <SettingsCard>
      {#if managed && runtime}<ManagedRuntimeControls
          {sidebar}
          workBusy={runtimeWorkBusy}
          {runtime}
          {instanceID}
          {disabled}
          onManage={onManageRuntime}
        />{:else if !managed}{@render modelControls()}{/if}
      {@render recognitionControls()}
    </SettingsCard>
  {:else if !managed}
    {@render modelControls()}
  {/if}
  {#if setup}
    <SettingsDisclosure
      title="Transcription options"
      description="Language, realtime, and recognition hints"
    >
      <div class="space-y-4 py-3">{@render optionalControls()}</div>
    </SettingsDisclosure>
  {:else if sidebar}
    {@render recognitionControls()}
    {#if onOpenAdvanced}<button
        type="button"
        class="flex min-h-7 w-full items-center justify-between gap-2 border-t border-hairline pt-1 text-left text-xs text-secondary-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        onclick={onOpenAdvanced}
        >More transcription options<ChevronRightIcon
          class="size-3 shrink-0"
          aria-hidden="true"
        /></button
      >{/if}
  {:else if draft}{@render finishingControls()}{:else}{@render optionalControls()}{/if}
  {#if !draft}<QuickSaveStatus
      quiet={sidebar}
      fields={["voice-transcription"]}
      pending={editor.quickSettingsPending}
      saved={editor.quickSettingsSaved}
      failed={editor.quickSettingsFailed}
    />{/if}
</div>

{#snippet modelControls()}
  <RuntimeModelPicker
    {sidebar}
    id={controlID("voice-model")}
    value={cfg.model}
    compact={!draft}
    immediate={!draft}
    showProfileName={false}
    models={availableModels}
    savedModels={rememberedModels(settings, Purpose.Voice).map((m) => m.model)}
    draftModels={draft ? editor.modelDraftIDs(Purpose.Voice) : []}
    serverLoaded={!!serverLoaded}
    disabled={busy || !connectionID}
    busy={testing}
    onChoose={chooseModel}
    onDiscover={test}
    onEnter={() => {
      void editor.ensureConnectionMetadata(Purpose.Voice, true);
    }}
    metadataStatus={editor.connectionMetadataStatus(Purpose.Voice)}
    onForget={draft
      ? () => {
          void editor.forgetModel(Purpose.Voice);
        }
      : undefined}
  />
  {#if !sidebar}<ModelProfilePicker
      id={controlID("voice-profile")}
      value={cfg.modelProfile}
      profiles={settings.modelProfiles.voiceTranscription ?? []}
      disabled={busy}
      compact={!draft}
      onChange={chooseProfile}
    />{/if}
{/snippet}

{#snippet optionalControls()}
  {@render recognitionControls()}
  {@render finishingControls()}
{/snippet}

{#snippet recognitionControls()}
  {#if profileNotice}<p
      class={draft
        ? "py-3 text-xs text-muted-foreground"
        : "text-xs text-muted-foreground"}
      role="status"
    >
      {profileNotice}
    </p>{/if}
  {#if sidebar ? realtimeAvailable || cfg.realtime : profile?.capabilities.realtime}
    <div
      class={draft
        ? "flex items-center justify-between gap-4 py-3"
        : sidebar
          ? "flex min-h-7 items-center justify-between gap-3"
          : "flex items-center justify-between gap-3 border-t border-hairline pt-3"}
    >
      <div>
        <label
          for={controlID("voice-realtime")}
          class="text-[13px] font-semibold"
          >{sidebar ? "Live dictation" : "Realtime transcription"}</label
        >
        {#if !sidebar}<p class="mt-1 text-xs text-muted-foreground">
            Show words as you speak, using this connection and model.
          </p>{/if}
      </div>
      <Switch
        id={controlID("voice-realtime")}
        bind:checked={() => cfg.realtime, setRealtime}
        disabled={busy ||
          (sidebar && !cfg.realtime && !realtimeAvailable) ||
          (!managed && !cfg.realtime && (!connectionID || !cfg.model))}
      />
    </div>
  {/if}
  {#if cfg.realtime && cfg.modelProfile === ID.Qwen3ASR}
    <p
      class={draft
        ? "py-3 text-xs leading-relaxed text-muted-foreground"
        : "text-xs leading-relaxed text-muted-foreground"}
    >
      {#if sidebar}Language is detected automatically in live dictation.
      {:else}Qwen realtime uses automatic language detection. Language, context,
        vocabulary, and temperature hints apply only with realtime off.{/if}
    </p>
  {/if}
  {#if (profile?.capabilities.languageHint || profile?.languages?.length) && (!cfg.realtime || profile?.realtimeLanguageHint)}
    <div class={draft ? "space-y-2 py-3" : "space-y-1.5"}>
      <label for={controlID("voice-language")} class="text-[13px] font-semibold"
        >Spoken language</label
      >
      <LanguagePicker
        id={controlID("voice-language")}
        immediate={!draft}
        restricted={!!profile.languages?.length}
        languages={profile.languages?.length
          ? profile.languages
          : (settings.transcriptionLanguages ?? [])}
        disabled={busy}
        bind:value={() => cfg.language, (language) => update({ language })}
      />
    </div>
  {/if}
  {#if !sidebar && !cfg.realtime && profile?.capabilities.transcriptionPrompt}
    <div class={draft ? "space-y-2 py-3" : "space-y-1.5"}>
      <label for={controlID("voice-prompt")} class="text-[13px] font-semibold"
        >Context hint</label
      ><Textarea
        id={controlID("voice-prompt")}
        rows={2}
        maxlength={8192}
        disabled={busy}
        value={cfg.transcriptionOptions.prompt}
        onchange={(event) =>
          update({
            transcriptionOptions: {
              ...cfg.transcriptionOptions,
              prompt: event.currentTarget.value,
            },
          })}
      />
    </div>
  {/if}
{/snippet}

{#snippet finishingControls()}
  <VocabularyLink {settings} voice />
  {#if draft && !cfg.realtime}
    <SettingsDisclosure
      title="Request settings"
      description="Timeout and supported temperature controls"
    >
      <div class="space-y-4 py-3">{@render requestControls()}</div>
    </SettingsDisclosure>
  {:else}
    {@render requestControls()}
  {/if}
  {#if cfg.realtime}
    <div class="flex items-center justify-between gap-3">
      <label for={controlID("voice-captions")} class="text-[13px] font-semibold"
        >Live overlay captions</label
      >
      <Switch
        id={controlID("voice-captions")}
        checked={cfg.captions}
        disabled={busy}
        onCheckedChange={(captions) => update({ captions })}
      />
    </div>
    <p
      class="border-t border-hairline pt-3 text-xs leading-relaxed text-muted-foreground"
    >
      Microphone audio streams while recording. Stop to finalize, clean up, and
      insert. Live mode uses the recording limit; silence trimming, checkpoints,
      and automatic stop apply to completed transcription.
    </p>
  {/if}
{/snippet}

{#snippet requestControls()}
  {#if !cfg.realtime && profile?.capabilities.transcriptionTemperature}
    <div class="flex items-center justify-between gap-3">
      <label
        for={controlID("voice-temperature-override")}
        class="text-[13px] font-semibold">Override temperature</label
      ><Switch
        id={controlID("voice-temperature-override")}
        checked={cfg.transcriptionOptions.temperatureOverride}
        disabled={busy}
        onCheckedChange={(temperatureOverride) =>
          update({
            transcriptionOptions: {
              ...cfg.transcriptionOptions,
              temperatureOverride,
            },
          })}
      />
    </div>
    {#if cfg.transcriptionOptions.temperatureOverride}<Input
        aria-label="Temperature"
        type="number"
        min="0"
        max="1"
        step="0.1"
        disabled={busy}
        value={cfg.transcriptionOptions.temperature}
        onchange={(event) =>
          update({
            transcriptionOptions: {
              ...cfg.transcriptionOptions,
              temperature: event.currentTarget.valueAsNumber,
            },
          })}
      />{/if}
  {/if}
  {#if draft && !cfg.realtime}
    <div class="space-y-1.5">
      <label for={controlID("voice-timeout")} class="text-[13px] font-semibold"
        >Recording request timeout (seconds)</label
      ><Input
        id={controlID("voice-timeout")}
        type="number"
        min="10"
        max="3600"
        step="10"
        disabled={busy}
        value={cfg.timeoutSeconds}
        onchange={(event) =>
          update({ timeoutSeconds: event.currentTarget.valueAsNumber })}
      />
      <p class="text-xs text-muted-foreground">
        Each completed recording or checkpoint gets this request budget.
      </p>
    </div>
  {/if}
{/snippet}
