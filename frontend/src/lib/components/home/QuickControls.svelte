<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import QuickToggle from "$lib/components/home/QuickToggle.svelte";
  import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import type { Device, Settings } from "$lib/state";
  import type { QuickSettingsField, QuickSettingsPatch } from "$lib/stores/editor.svelte";
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
    section = "all",
    devices,
    pending = [],
    savedField = null,
    onUpdate,
    onOpenAudioSettings,
    onOpenDeliverySettings,
    disabled = false,
  }: {
    settings: Settings;
    section?: "all" | "audio" | "delivery";
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

  const isPending = (field: QuickSettingsField) => pending.includes(field);
</script>

<span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
  {toolbarAnnouncement}
</span>

<!-- Capture and Delivery are one control surface, separated by a real internal
     rule rather than nested cards. They stay open because these are the rack's
     immediate behavior controls; the longer endpoint modules fold below it. -->
<section
  class="@container shrink-0 overflow-hidden"
  class:framed={section === "all"}
  aria-label="Capture and delivery"
>
  {#if section !== "delivery"}
    <div class="control-group" class:embedded={section === "audio"}>
      <div class="group-head">
        <h2 class="text-sm font-semibold">Audio</h2>
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

      <div class="grid grid-cols-1 gap-x-6 gap-y-1 @min-[460px]:grid-cols-2">
        <div class="col-span-full mb-2">
          <DropdownMenu.Root>
            <DropdownMenu.Trigger disabled={controlsDisabled}>
              {#snippet child({ props })}
                <button
                  {...props}
                  type="button"
                  class="microphone-control"
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
        </div>

        <QuickToggle
          label="Voice detection"
          checked={vadEnabled}
          disabled={controlsDisabled}
          pending={isPending("vad-enabled")}
          saved={savedField === "vad-enabled"}
          onChange={toggleVAD}
        />
        <QuickToggle
          label="Checkpoints"
          checked={checkpointsEnabled}
          disabled={controlsDisabled}
          pending={isPending("silence-splitting")}
          saved={savedField === "silence-splitting"}
          onChange={toggleCheckpoints}
        />
        <QuickToggle
          label="Overlay"
          checked={overlayEnabled}
          disabled={controlsDisabled}
          pending={isPending("overlay-enabled")}
          saved={savedField === "overlay-enabled"}
          onChange={toggleOverlay}
        />
      </div>
    </div>
  {/if}
  {#if section !== "audio"}
    <div
      class="control-group"
      class:embedded={section === "delivery"}
      class:divided={section === "all"}
    >
      <div class="group-head">
        <h2 class="text-sm font-semibold">Delivery</h2>
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

      <div class="grid grid-cols-1 gap-x-6 gap-y-1 @min-[460px]:grid-cols-2">
        <QuickToggle
          label="Direct input"
          checked={directInputEnabled}
          disabled={controlsDisabled}
          pending={isPending("delivery")}
          saved={savedField === "delivery"}
          onChange={toggleDelivery}
        />
        <QuickToggle
          label="Keep history"
          checked={historyEnabled}
          disabled={controlsDisabled}
          pending={isPending("history-enabled")}
          saved={savedField === "history-enabled"}
          onChange={toggleHistory}
        />
      </div>
      <p class="mt-2 text-xs text-muted-foreground">Direct input off uses manual copy.</p>
    </div>
  {/if}
</section>

<style>
  .control-group {
    padding: 1rem 0;
  }
  .framed {
    border-bottom: 1px solid var(--hairline);
  }
  .embedded {
    padding: 0;
  }
  .divided {
    border-top: 1px solid var(--hairline);
  }
  .group-head {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .door {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border-radius: var(--radius-sm);
    color: var(--secondary-foreground);
  }
  .door:hover {
    background: var(--subtle-fill-hover);
  }
  .door:focus-visible,
  .microphone-control:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 2px;
  }
  .microphone-control {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    min-height: 2.25rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--input);
    border-radius: var(--radius-md);
    background: var(--well);
    font-size: 0.8125rem;
  }
  .microphone-control:disabled {
    opacity: 0.5;
  }
</style>
