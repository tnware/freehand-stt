<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import type { Device, Settings } from "$lib/state";
  import type { QuickSettingsField, QuickSettingsPatch } from "$lib/stores/session.svelte";
  import {
    SYSTEM_DEFAULT_LABEL,
    SYSTEM_DEFAULT_MICROPHONE,
    microphoneChoiceFor,
    microphoneIDFor,
    microphoneLabel,
    microphoneMissing,
  } from "$lib/utils/microphone";
  import { cn } from "$lib/utils";

  let {
    settings,
    devices,
    pending = [],
    savedField = null,
    onUpdate,
    onOpenAudioSettings,
    onOpenDeliverySettings,
    disabled = false,
  }: {
    settings: Settings;
    devices: Device[];
    pending?: QuickSettingsField[];
    savedField?: QuickSettingsField | null;
    onUpdate: (patch: QuickSettingsPatch, field: QuickSettingsField) => Promise<boolean>;
    onOpenAudioSettings: () => void;
    onOpenDeliverySettings: () => void;
    disabled?: boolean;
  } = $props();

  let selectedMicrophone = $state(SYSTEM_DEFAULT_MICROPHONE);
  let vadEnabled = $state(false);
  let checkpointsEnabled = $state(false);
  let directInputEnabled = $state(false);
  let historyEnabled = $state(false);
  let overlayEnabled = $state(false);

  $effect(() => {
    selectedMicrophone = microphoneChoiceFor(settings.microphoneID);
    vadEnabled = settings.vadEnabled;
    checkpointsEnabled = settings.silenceSplitting;
    directInputEnabled = settings.autoInsert;
    historyEnabled = settings.historyEnabled;
    overlayEnabled = settings.overlayEnabled;
  });

  const saving = $derived(pending.length > 0);
  const controlsDisabled = $derived(disabled || saving);
  const selectedMicrophoneLabel = $derived(microphoneLabel(selectedMicrophone, devices));
  const selectedMicrophoneMissing = $derived(microphoneMissing(selectedMicrophone, devices));
  const toolbarFields: QuickSettingsField[] = [
    "microphone",
    "vad-enabled",
    "silence-splitting",
    "delivery",
    "history-enabled",
    "overlay-enabled",
  ];
  const toolbarAnnouncement = $derived.by(() => {
    const pendingField = toolbarFields.find((field) => pending.includes(field));
    if (pendingField) return "Saving quick control.";
    if (!savedField || !toolbarFields.includes(savedField)) return "";
    return "Quick control saved.";
  });

  async function chooseMicrophone(choice: string) {
    if (!choice || choice === microphoneChoiceFor(settings.microphoneID)) return;
    selectedMicrophone = choice;
    if (!(await onUpdate({ microphoneID: microphoneIDFor(choice) }, "microphone"))) {
      selectedMicrophone = microphoneChoiceFor(settings.microphoneID);
    }
  }

  async function toggleVAD(enabled: boolean) {
    vadEnabled = enabled;
    const patch: QuickSettingsPatch = { vadEnabled: enabled };
    if (!enabled) {
      checkpointsEnabled = false;
      patch.silenceTrimming = false;
      patch.autoStopEnabled = false;
      patch.silenceSplitting = false;
      if (settings.maxDurationSeconds > 262) patch.maxDurationSeconds = 262;
    }
    if (!(await onUpdate(patch, "vad-enabled"))) {
      vadEnabled = settings.vadEnabled;
      checkpointsEnabled = settings.silenceSplitting;
    }
  }

  async function toggleCheckpoints(enabled: boolean) {
    checkpointsEnabled = enabled;
    const patch: QuickSettingsPatch = { silenceSplitting: enabled };
    if (enabled) {
      vadEnabled = true;
      patch.vadEnabled = true;
    } else if (settings.maxDurationSeconds > 262) {
      patch.maxDurationSeconds = 262;
    }
    if (!(await onUpdate(patch, "silence-splitting"))) {
      vadEnabled = settings.vadEnabled;
      checkpointsEnabled = settings.silenceSplitting;
    }
  }

  async function toggleDelivery(enabled: boolean) {
    directInputEnabled = enabled;
    if (!(await onUpdate({ autoInsert: enabled }, "delivery"))) {
      directInputEnabled = settings.autoInsert;
    }
  }

  async function toggleHistory(enabled: boolean) {
    historyEnabled = enabled;
    if (!(await onUpdate({ historyEnabled: enabled }, "history-enabled"))) {
      historyEnabled = settings.historyEnabled;
    }
  }

  async function toggleOverlay(enabled: boolean) {
    overlayEnabled = enabled;
    if (!(await onUpdate({ overlayEnabled: enabled }, "overlay-enabled"))) {
      overlayEnabled = settings.overlayEnabled;
    }
  }

  const lamp = (on: boolean) => (on ? "bg-primary" : "bg-border");
  const isPending = (field: QuickSettingsField) => pending.includes(field);
</script>

<span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
  {toolbarAnnouncement}
</span>

<!-- Capture and Delivery are one control surface, separated by a real internal
     rule rather than nested cards. They stay open because these are the rack's
     immediate behavior controls; the longer endpoint modules fold below it. -->
<section
  class="shrink-0 overflow-hidden rounded-lg border border-card-stroke bg-card shadow-lift"
  aria-label="Capture and delivery"
>
  <div class="control-group">
    <div class="group-head">
      <h2 class="caption">Capture</h2>
      <span class="flex-1"></span>
      <button
        type="button"
        class="door"
        aria-label="Open audio settings"
        title="Open audio settings"
        onclick={onOpenAudioSettings}
      >
        <SlidersIcon class="size-[13px]" />
      </button>
    </div>

    <div class="grid grid-cols-2 gap-x-3.5 gap-y-2">
    <DropdownMenu.Root>
      <DropdownMenu.Trigger disabled={controlsDisabled}>
        {#snippet child({ props })}
          <button
            {...props}
            type="button"
            class="lamp-row"
            aria-label={`Microphone: ${selectedMicrophoneLabel}`}
            title={`Microphone: ${selectedMicrophoneLabel}`}
          >
            <span
              class={cn(
                "size-1.5 shrink-0 rounded-full",
                selectedMicrophoneMissing ? "bg-warning" : "bg-success",
              )}
            ></span>
            <span class="min-w-0 flex-1 truncate text-left">{selectedMicrophoneLabel}</span>
            <ChevronDownIcon class="size-3 shrink-0 text-ink-quiet" />
          </button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="start" class="w-72">
        <DropdownMenu.Group>
          <DropdownMenu.GroupHeading>Microphone</DropdownMenu.GroupHeading>
          <DropdownMenu.RadioGroup
            bind:value={selectedMicrophone}
            onValueChange={(choice) => void chooseMicrophone(choice)}
          >
            <DropdownMenu.RadioItem value={SYSTEM_DEFAULT_MICROPHONE}>
              {SYSTEM_DEFAULT_LABEL}
            </DropdownMenu.RadioItem>
            {#if selectedMicrophoneMissing}
              <DropdownMenu.RadioItem value={selectedMicrophone}>
                {selectedMicrophoneLabel}
              </DropdownMenu.RadioItem>
            {/if}
            {#each devices as device (device.id)}
              <DropdownMenu.RadioItem value={device.id}>{device.name}</DropdownMenu.RadioItem>
            {/each}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Group>
      </DropdownMenu.Content>
    </DropdownMenu.Root>

    <button
      type="button"
      class="lamp-row"
      role="switch"
      aria-checked={vadEnabled}
      disabled={controlsDisabled}
      onclick={() => void toggleVAD(!vadEnabled)}
    >
      <span class="size-1.5 shrink-0 rounded-full {lamp(vadEnabled)}"></span>
      <span class="min-w-0 flex-1 truncate text-left">Voice detection</span>
      {#if isPending("vad-enabled")}
        <LoaderCircleIcon class="size-3 shrink-0 animate-spin text-ink-quiet" />
      {:else if savedField === "vad-enabled"}
        <CheckIcon class="size-3 shrink-0 text-success" />
      {:else}
        <span class="figure shrink-0 text-[9.5px] text-ink-quiet">
          {vadEnabled ? "ON" : "OFF"}
        </span>
      {/if}
    </button>

    <button
      type="button"
      class="lamp-row"
      role="switch"
      aria-checked={checkpointsEnabled}
      disabled={controlsDisabled}
      onclick={() => void toggleCheckpoints(!checkpointsEnabled)}
    >
      <span class="size-1.5 shrink-0 rounded-full {lamp(checkpointsEnabled)}"></span>
      <span class="min-w-0 flex-1 truncate text-left">Checkpoints</span>
      {#if isPending("silence-splitting")}
        <LoaderCircleIcon class="size-3 shrink-0 animate-spin text-ink-quiet" />
      {:else if savedField === "silence-splitting"}
        <CheckIcon class="size-3 shrink-0 text-success" />
      {:else}
        <span class="figure shrink-0 text-[9.5px] text-ink-quiet">
          {checkpointsEnabled ? "ON" : "OFF"}
        </span>
      {/if}
    </button>

    <button
      type="button"
      class="lamp-row"
      role="switch"
      aria-checked={overlayEnabled}
      disabled={controlsDisabled}
      onclick={() => void toggleOverlay(!overlayEnabled)}
    >
      <span class="size-1.5 shrink-0 rounded-full {lamp(overlayEnabled)}"></span>
      <span class="min-w-0 flex-1 truncate text-left">Overlay</span>
      {#if isPending("overlay-enabled")}
        <LoaderCircleIcon class="size-3 shrink-0 animate-spin text-ink-quiet" />
      {:else if savedField === "overlay-enabled"}
        <CheckIcon class="size-3 shrink-0 text-success" />
      {:else}
        <span class="figure shrink-0 text-[9.5px] text-ink-quiet">
          {overlayEnabled ? "ON" : "OFF"}
        </span>
      {/if}
    </button>
    </div>
  </div>

  <div class="control-group border-t border-hairline">
    <div class="group-head">
      <h2 class="caption">Delivery</h2>
      <span class="flex-1"></span>
      <button
        type="button"
        class="door"
        aria-label="Open general settings"
        title="Open general settings"
        onclick={onOpenDeliverySettings}
      >
        <SlidersIcon class="size-[13px]" />
      </button>
    </div>

    <div class="grid grid-cols-2 gap-x-3.5 gap-y-2">
    <button
      type="button"
      class="lamp-row"
      role="switch"
      aria-checked={directInputEnabled}
      disabled={controlsDisabled}
      onclick={() => void toggleDelivery(!directInputEnabled)}
    >
      <span class="size-1.5 shrink-0 rounded-full {lamp(directInputEnabled)}"></span>
      <span class="min-w-0 flex-1 truncate text-left">
        {directInputEnabled ? "Typed straight in" : "Manual copy"}
      </span>
      {#if isPending("delivery")}
        <LoaderCircleIcon class="size-3 shrink-0 animate-spin text-ink-quiet" />
      {:else if savedField === "delivery"}
        <CheckIcon class="size-3 shrink-0 text-success" />
      {:else}
        <span class="figure shrink-0 text-[9.5px] text-ink-quiet">
          {directInputEnabled ? "ON" : "OFF"}
        </span>
      {/if}
    </button>

    <button
      type="button"
      class="lamp-row"
      role="switch"
      aria-checked={historyEnabled}
      disabled={controlsDisabled}
      onclick={() => void toggleHistory(!historyEnabled)}
    >
      <span class="size-1.5 shrink-0 rounded-full {lamp(historyEnabled)}"></span>
      <span class="min-w-0 flex-1 truncate text-left">Keep history</span>
      {#if isPending("history-enabled")}
        <LoaderCircleIcon class="size-3 shrink-0 animate-spin text-ink-quiet" />
      {:else if savedField === "history-enabled"}
        <CheckIcon class="size-3 shrink-0 text-success" />
      {:else}
        <span class="figure shrink-0 text-[9.5px] text-ink-quiet">
          {historyEnabled ? "ON" : "OFF"}
        </span>
      {/if}
    </button>
    </div>
  </div>
</section>

<style>
  .control-group {
    padding: 0.75rem;
  }
  .group-head {
    display: flex;
    height: 1.25rem;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.625rem;
  }
  .door {
    display: grid;
    place-items: center;
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    color: var(--ink-quiet);
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .door:hover {
    background-color: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  .door:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
  .lamp-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    padding: 0.125rem 0.25rem;
    margin: -0.125rem -0.25rem;
    border-radius: var(--radius-sm);
    font-size: 0.719rem;
    color: var(--card-foreground);
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .lamp-row:not(:disabled):hover {
    background-color: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  .lamp-row[aria-checked="false"] {
    color: var(--ink-quiet);
  }
  .lamp-row:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .lamp-row:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
</style>
