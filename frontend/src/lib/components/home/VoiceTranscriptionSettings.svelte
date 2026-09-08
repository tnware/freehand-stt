<script lang="ts">
  import SettingsDisclosure from "../settings/SettingsDisclosure.svelte";
  import RuntimeModelPicker from "../settings/RuntimeModelPicker.svelte";
  import QuickSaveStatus from "../settings/QuickSaveStatus.svelte";
  import VocabularyLink from "$lib/components/settings/VocabularyLink.svelte";
  import ModelProfilePicker from "../settings/ModelProfilePicker.svelte";
  import { onDestroy } from "svelte";
  import { ID } from "$bindings/modelprofile";
  import LanguagePicker from "$lib/components/settings/LanguagePicker.svelte";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import { Purpose } from "$bindings/savedconnection";
  import { Service as ConnectionService } from "$bindings/connection";
  import type { Settings } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Switch } from "$lib/components/ui/switch";

  let {
    editor,
    settings,
    disabled,
    draft = false,
    setup = false,
    onAddConnection,
  }: {
    editor: SettingsEditor;
    settings: Settings;
    disabled: boolean;
    draft?: boolean;
    setup?: boolean;
    onAddConnection: (purpose: Purpose) => void;
  } = $props();
  const cfg = $derived(settings.voiceTranscription);
  let profileNotice = $state("");
  const backend = $derived(
    settings.compatibilityProfiles.transcription?.find((p) => p.id === cfg.compatibilityProfile),
  );
  const serverLoaded = $derived(
    backend?.capabilities.serverLoadedModel && !backend?.capabilities.realtime,
  );
  const profile = $derived(
    settings.modelProfiles.voiceTranscription?.find((p) => p.id === cfg.modelProfile),
  );
  function chooseProfile(value: string) {
    const selected = settings.modelProfiles.voiceTranscription?.find((p) => p.id === value);
    if (!selected || selected.id === cfg.modelProfile) return;
    const language =
      selected.languages?.length && !selected.languages.some((l) => l.code === cfg.language)
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
      options:
        selected.id === ID.Nemotron35 ? { vocabulary: "", boost: 3 } : { vocabulary: "", boost: 0 },
      transcriptionOptions: {
        prompt: "",
        hotwords: "",
        temperatureOverride: false,
        temperature: 0,
      },
    });
  }
  const connectionID = $derived(settings.savedConnections.selected?.[Purpose.Voice] ?? "");
  const busy = $derived(
    disabled ||
      editor.saving ||
      editor.voiceConnectionTesting ||
      editor.isQuickSettingsPending("voice-transcription"),
  );
  let testing = $state(false);
  let testedID = $state("");
  let models = $state<string[]>([]);
  let message = $state("");
  let revision = 0;
  onDestroy(() => {
    revision++;
  });
  const availableModels = $derived(
    testedID === connectionID ? models : (editor.currentVoiceConnection?.modelIDs ?? []),
  );
  function chooseModel(model: string) {
    return draft
      ? editor.chooseModel(Purpose.Voice, model)
      : editor.updateQuickSettings({ voiceTranscription: { model } }, "voice-transcription");
  }
  function update(patch: Partial<Settings["voiceTranscription"]>) {
    if (draft) {
      if (patch.model !== undefined) editor.chooseModel(Purpose.Voice, patch.model);
      Object.assign(settings.voiceTranscription, patch);
    } else void editor.updateQuickSettings({ voiceTranscription: patch }, "voice-transcription");
  }
  async function test() {
    const id = connectionID;
    const current = ++revision;
    testing = true;
    message = "";
    try {
      const result = draft
        ? await ConnectionService.TestSavedConnection(id)
        : await editor.testVoiceConnection();
      if (!result) return;
      if (current !== revision || id !== connectionID) return;
      testedID = id;
      models = result.modelIDs ?? [];
      message = result.reachable
        ? models.length
          ? "Connected. Choose the model loaded by this server."
          : "Connected, but no loaded model was reported."
        : "Could not reach the server. Check its address and credential in Connections.";
    } catch {
      if (current === revision && id === connectionID) {
        testedID = id;
        message = "Connection check failed.";
      }
    } finally {
      if (current === revision) testing = false;
    }
  }
</script>

<div class="space-y-4">
  {#if !draft}
    {#if !setup}<h3 class="text-sm font-semibold">Transcription</h3>{/if}
    <div class="space-y-1.5">
      <label for="voice-connection" class="text-xs font-medium">Connection</label>
      <div class="flex gap-2">
        <ConnectionSelect
          id="voice-connection"
          catalog={settings.savedConnections}
          purpose={Purpose.Voice}
          disabled={busy || testing}
          onChange={(change) => editor.changeConnection(change)}
          onAdd={() => onAddConnection(Purpose.Voice)}
        />
      </div>
    </div>
  {/if}
  {#if message && testedID === connectionID}<p class="text-xs text-muted-foreground" role="status">
      {message}
    </p>{/if}
  <RuntimeModelPicker
    id="voice-model"
    value={cfg.model}
    compact
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
    onForget={draft
      ? () => {
          void editor.forgetModel(Purpose.Voice);
        }
      : undefined}
  />
  <ModelProfilePicker
    id="voice-profile"
    value={cfg.modelProfile}
    profiles={settings.modelProfiles.voiceTranscription ?? []}
    disabled={busy}
    compact
    onChange={chooseProfile}
  />
  {#if setup}
    <SettingsDisclosure
      title="Transcription options"
      description="Language, realtime, and recognition hints"
    >
      <div class="space-y-4 p-4">{@render optionalControls()}</div>
    </SettingsDisclosure>
  {:else}{@render optionalControls()}{/if}
  {#if !draft}<QuickSaveStatus
      fields={["voice-transcription"]}
      pending={editor.quickSettingsPending}
      saved={editor.quickSettingsSaved}
      failed={editor.quickSettingsFailed}
    />{/if}
</div>

{#snippet optionalControls()}
  {#if profileNotice}<p class="text-xs text-muted-foreground" role="status">
      {profileNotice}
    </p>{/if}
  {#if profile?.capabilities.realtime}
    <div class="flex items-center justify-between gap-3 border-t border-hairline pt-3">
      <div>
        <label for="voice-realtime" class="text-sm font-medium">Realtime transcription</label>
        <p class="mt-1 text-xs text-muted-foreground">
          Show words as you speak, using this connection and model.
        </p>
      </div>
      <Switch
        id="voice-realtime"
        checked={cfg.realtime}
        disabled={busy || (!cfg.realtime && (!connectionID || !cfg.model))}
        onCheckedChange={(realtime) => update({ realtime })}
      />
    </div>
  {/if}
  {#if cfg.realtime && cfg.modelProfile === ID.Qwen3ASR}
    <p class="text-xs leading-relaxed text-muted-foreground">
      Qwen realtime uses automatic language detection. Language, context, vocabulary, and
      temperature hints apply only with realtime off.
    </p>
  {/if}
  {#if (profile?.capabilities.languageHint || profile?.languages?.length) && (!cfg.realtime || profile?.realtimeLanguageHint)}
    <div class="space-y-1.5">
      <label for="voice-language" class="text-sm font-medium">Spoken language</label>
      <LanguagePicker
        id="voice-language"
        restricted={!!profile.languages?.length}
        languages={profile.languages?.length
          ? profile.languages
          : (settings.transcriptionLanguages ?? [])}
        disabled={busy}
        bind:value={() => cfg.language, (language) => update({ language })}
      />
    </div>
  {/if}
  {#if !cfg.realtime && profile?.capabilities.transcriptionPrompt}
    <div class="space-y-1.5">
      <label for="voice-prompt" class="text-sm font-medium">Context hint</label><textarea
        id="voice-prompt"
        rows="2"
        maxlength="8192"
        class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        disabled={busy}
        value={cfg.transcriptionOptions.prompt}
        onchange={(event) =>
          update({
            transcriptionOptions: {
              ...cfg.transcriptionOptions,
              prompt: event.currentTarget.value,
            },
          })}
      ></textarea>
    </div>
  {/if}
  <VocabularyLink {settings} voice />
  {#if draft && !cfg.realtime}
    <SettingsDisclosure
      title="Request settings"
      description="Timeout and supported temperature controls"
    >
      <div class="space-y-4 px-5 py-4">{@render requestControls()}</div>
    </SettingsDisclosure>
  {:else}
    {@render requestControls()}
  {/if}
  {#if cfg.realtime}
    <div class="flex items-center justify-between gap-3">
      <label for="voice-captions" class="text-sm">Live overlay captions</label>
      <Switch
        id="voice-captions"
        checked={cfg.captions}
        disabled={busy}
        onCheckedChange={(captions) => update({ captions })}
      />
    </div>
    <p class="border-t border-hairline pt-3 text-xs leading-relaxed text-muted-foreground">
      Microphone audio streams while recording. Stop to finalize, clean up, and insert. Live mode
      uses the recording limit; silence trimming, checkpoints, and automatic stop apply to completed
      transcription.
    </p>
  {/if}
{/snippet}

{#snippet requestControls()}
  {#if !cfg.realtime && profile?.capabilities.transcriptionTemperature}
    <div class="flex items-center justify-between gap-3">
      <label for="voice-temperature-override" class="text-sm">Override temperature</label><Switch
        id="voice-temperature-override"
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
    {#if cfg.transcriptionOptions.temperatureOverride}<input
        aria-label="Temperature"
        type="number"
        min="0"
        max="1"
        step="0.1"
        class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
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
      <label for="voice-timeout" class="text-xs font-medium"
        >Recording request timeout (seconds)</label
      ><input
        id="voice-timeout"
        type="number"
        min="10"
        max="3600"
        step="10"
        class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
        disabled={busy}
        value={cfg.timeoutSeconds}
        onchange={(event) => update({ timeoutSeconds: event.currentTarget.valueAsNumber })}
      />
      <p class="text-xs text-muted-foreground">
        Each completed recording or checkpoint gets this request budget.
      </p>
    </div>
  {/if}
{/snippet}
