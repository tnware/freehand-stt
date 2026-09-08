<script lang="ts">
  import { onDestroy } from "svelte";
  import { Purpose } from "$bindings/savedconnection";
  import { Service as ConnectionService } from "$bindings/connection";
  import type { Settings } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import * as Select from "$lib/components/ui/select";

  let {
    editor,
    settings,
    disabled,
    draft = false,
    onAddConnection,
  }: {
    editor: SettingsEditor;
    settings: Settings;
    disabled: boolean;
    draft?: boolean;
    onAddConnection: (purpose: Purpose) => void;
  } = $props();
  const cfg = $derived(settings.realtime);
  const connectionID = $derived(settings.savedConnections.selected?.[Purpose.Realtime] ?? "");
  const busy = $derived(disabled || editor.saving || editor.isQuickSettingsPending("realtime"));
  let testing = $state(false);
  let testedID = $state("");
  let models = $state<string[]>([]);
  let message = $state("");
  let revision = 0;
  onDestroy(() => {
    revision++;
  });
  const availableModels = $derived([
    ...new Set([cfg.model, ...(testedID === connectionID ? models : [])].filter(Boolean)),
  ]);
  function update(patch: Partial<Settings["realtime"]>) {
    if (draft) {
      if (patch.model !== undefined) editor.chooseModel(Purpose.Realtime, patch.model);
      Object.assign(settings.realtime, patch);
    } else void editor.updateQuickSettings({ realtime: patch }, "realtime");
  }
  async function test() {
    const id = connectionID;
    const current = ++revision;
    testing = true;
    message = "";
    try {
      const result = await ConnectionService.TestSavedConnection(id);
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
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-sm font-semibold">Live transcription</h3>
      <p class="mt-1 text-xs text-muted-foreground">Nemotron 3.5 · NeMo-Speech.cpp</p>
    </div>
    <Switch
      aria-label="Use live transcription"
      checked={cfg.enabled}
      disabled={busy || (!cfg.enabled && (!connectionID || !cfg.model))}
      onCheckedChange={(enabled) => update({ enabled })}
    />
  </div>
  <div class="space-y-1.5">
    <label for="realtime-connection" class="text-xs font-medium">Connection</label>
    <div class="flex gap-2">
      <ConnectionSelect
        id="realtime-connection"
        catalog={settings.savedConnections}
        purpose={Purpose.Realtime}
        disabled={busy || testing}
        onChange={(change) => editor.changeConnection(change)}
        onAdd={() => onAddConnection(Purpose.Realtime)}
      />
      <Button
        variant="outline"
        size="sm"
        disabled={busy || testing || !connectionID}
        onclick={test}
      >
        {testing ? "Checking…" : "Check"}
      </Button>
    </div>
  </div>
  {#if message && testedID === connectionID}<p class="text-xs text-muted-foreground" role="status">
      {message}
    </p>{/if}
  <div class="space-y-1.5">
    <label for="realtime-model" class="text-xs font-medium">Loaded model</label>
    <Select.Root
      type="single"
      value={cfg.model}
      disabled={busy || !connectionID}
      onValueChange={(model) => update({ model })}
    >
      <Select.Trigger id="realtime-model" class="w-full"
        ><span class="truncate">{cfg.model || "Check the connection to find its model"}</span
        ></Select.Trigger
      >
      <Select.Content
        >{#each availableModels as model (model)}<Select.Item value={model} label={model}
            >{model}</Select.Item
          >{/each}</Select.Content
      >
    </Select.Root>
    <p class="text-xs text-muted-foreground">
      Use with Nemotron 3.5 ASR streaming 0.6B. Model names do not verify model behavior.
    </p>
  </div>
  <div class="space-y-1.5">
    <label for="realtime-language" class="text-xs font-medium">Spoken language</label>
    <Select.Root
      type="single"
      value={cfg.language}
      disabled={busy}
      onValueChange={(language) => update({ language })}
    >
      <Select.Trigger id="realtime-language" class="w-full"
        >{settings.realtimeLanguages?.find((item) => item.code === cfg.language)?.label ??
          cfg.language}</Select.Trigger
      >
      <Select.Content
        >{#each settings.realtimeLanguages ?? [] as language (language.code)}<Select.Item
            value={language.code}
            label={language.label}>{language.label}</Select.Item
          >{/each}</Select.Content
      >
    </Select.Root>
  </div>
  <div class="space-y-1.5">
    <label for="realtime-vocabulary" class="text-xs font-medium">Vocabulary hints</label>
    <textarea
      id="realtime-vocabulary"
      rows="3"
      maxlength="2048"
      disabled={busy}
      class="w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      placeholder="One name or phrase per line"
      value={cfg.options.vocabulary}
      onchange={(event) =>
        update({ options: { ...cfg.options, vocabulary: event.currentTarget.value } })}
    ></textarea>
    <p class="text-xs text-muted-foreground">
      Names and terminology, up to 32 phrases. For rewrite instructions, use Cleanup.
    </p>
  </div>
  <div class="flex items-center justify-between gap-3">
    <label for="realtime-boost" class="text-xs font-medium">Vocabulary strength</label>
    <input
      id="realtime-boost"
      type="number"
      min="0"
      max="5"
      step="0.5"
      value={cfg.options.boost}
      disabled={busy}
      class="h-8 w-20 rounded-md border border-input bg-background px-2 text-sm"
      onchange={(event) =>
        update({ options: { ...cfg.options, boost: event.currentTarget.valueAsNumber } })}
    />
  </div>
  <div class="flex items-center justify-between gap-3">
    <label for="realtime-captions" class="text-sm">Live overlay captions</label>
    <Switch
      id="realtime-captions"
      checked={cfg.captions}
      disabled={busy}
      onCheckedChange={(captions) => update({ captions })}
    />
  </div>
  <p class="border-t border-hairline pt-3 text-xs leading-relaxed text-muted-foreground">
    Microphone audio streams while recording. Stop to finalize, clean up, and insert. Live mode uses
    the recording limit; silence trimming, checkpoints, and automatic stop apply to completed
    transcription.
  </p>
</div>
