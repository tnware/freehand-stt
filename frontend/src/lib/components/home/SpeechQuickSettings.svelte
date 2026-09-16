<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import QuickSaveStatus from "../settings/QuickSaveStatus.svelte";
  import SpeechModelControls from "../settings/SpeechModelControls.svelte";
  import type { QuickSettingsPatch } from "$lib/stores/editor.svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Popover from "$lib/components/ui/popover";
  import { Purpose, type Change } from "$bindings/savedconnection";
  import type { Settings } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import { cn } from "$lib/utils";

  let {
    settings,
    runtime,
    runtimeWorkBusy = false,
    onManageRuntime = () => {},
    editor,
    disabled,
    sidebar = false,
    onAddConnection,
    onOpenSettings,
  }: {
    settings: Settings;
    runtime?: ManagedRuntimeState;
    runtimeWorkBusy?: boolean;
    onManageRuntime?: () => void;
    editor: SettingsEditor;
    disabled: boolean;
    sidebar?: boolean;
    onAddConnection: (purpose: Purpose) => void;
    onOpenSettings: () => void;
  } = $props();
  const uid = $props.id();
  const connectionID = uid + "-speech-connection";
  let open = $state(false);
  const current = $derived(editor.applied ?? settings);
  const speech = $derived(current.textToSpeech);
  const busy = $derived(
    disabled ||
      editor.saving ||
      editor.quickSettingsPending.length > 0 ||
      !editor.applied,
  );
  function update(textToSpeech: QuickSettingsPatch["textToSpeech"]) {
    if (busy || !editor.applied) return Promise.resolve(false);
    return editor.updateQuickSettings({ textToSpeech }, "speech-controls");
  }
  function changeConnection(change: Change) {
    if (busy || !editor.applied) return Promise.resolve(false);
    return editor.changeConnection(change);
  }
  function navigate(action: () => void) {
    if (disabled || editor.saving) return;
    open = false;
    action();
  }
</script>

<div
  class={sidebar
    ? "flex min-w-0 flex-col gap-2"
    : "flex min-w-0 items-center gap-2"}
  role="group"
  aria-label="Speech quick settings"
>
  {#if sidebar}
    {@render controls()}
  {:else}
    <Popover.Root bind:open>
      <Popover.Trigger
        {disabled}
        aria-label="Speech settings"
        title="Speech settings"
        class={cn(
          buttonVariants({ variant: "outline", size: "sm" }),
          "gap-2 px-2 text-secondary-foreground aria-expanded:bg-accent-wash aria-expanded:text-accent-text",
        )}
      >
        <ProviderIcon profile={speech.compatibilityProfile} size={16} />
        <span class="text-[13px]">Speech</span>
        <ChevronDownIcon class="size-3 text-muted-foreground" />
      </Popover.Trigger>
      <Popover.Content
        role="dialog"
        aria-label="Speech settings"
        class="space-y-4"
      >
        <h3 class="text-sm font-semibold">Text to speech</h3>
        {@render controls()}
      </Popover.Content>
    </Popover.Root>
    <span
      class="hidden min-w-0 truncate text-xs text-muted-foreground @min-[540px]:inline"
      title={speech.voice || undefined}>{speech.voice || "Choose a voice"}</span
    >
  {/if}
</div>

{#snippet controls()}
  <div class="space-y-1.5">
    <label for={connectionID} class="content-value">Connection</label>
    <ConnectionSelect
      id={connectionID}
      catalog={current.savedConnections}
      runtimeInstances={runtime?.instances}
      purpose={Purpose.Speech}
      compact={sidebar}
      disabled={busy}
      onAdd={() => {
        if (!busy) navigate(() => onAddConnection(Purpose.Speech));
      }}
      onChange={changeConnection}
    />
  </div>
  <SpeechModelControls
    {runtimeWorkBusy}
    {runtime}
    onManageRuntime={() => navigate(onManageRuntime)}
    onEnter={() => {
      if (!busy) void editor.ensureConnectionMetadata(Purpose.Speech, true);
    }}
    metadataStatus={editor.connectionMetadataStatus(Purpose.Speech)}
    settings={current}
    compact
    immediate
    showAdvanced={!sidebar}
    busy={busy || !current.savedConnections.selected?.speech}
    models={editor.ttsConnectionStale ||
    editor.connectionResultStale(Purpose.Speech, current)
      ? []
      : (editor.ttsConnection?.modelIDs ?? [])}
    modelsBusy={editor.ttsConnectionTesting}
    voices={editor.voicesFor(current)}
    voicesBusy={editor.voicesBusy}
    onChooseModel={(model) => update({ model })}
    onVoice={(voice) => update({ voice })}
    onSpeed={(speed) => update({ speed })}
    onOptions={(options) => update({ options })}
    onDiscoverModels={() => {
      if (!busy) void editor.testAppliedConnection(Purpose.Speech);
    }}
    onDiscoverVoices={() => {
      if (!busy) void editor.discoverVoices(true);
    }}
  />
  <QuickSaveStatus
    quiet={sidebar}
    fields={["speech-controls"]}
    pending={editor.quickSettingsPending}
    saved={editor.quickSettingsSaved}
    failed={editor.quickSettingsFailed}
  />
  <Button
    variant={sidebar ? "ghost" : "outline"}
    size="sm"
    class={sidebar
      ? "w-full justify-start px-0 text-xs text-secondary-foreground"
      : "w-full"}
    disabled={disabled || editor.saving}
    onclick={() => navigate(onOpenSettings)}>All speech settings</Button
  >
{/snippet}
