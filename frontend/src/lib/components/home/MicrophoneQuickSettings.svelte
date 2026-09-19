<script lang="ts">
  import * as Select from "$lib/components/ui/select";
  import PickerRefreshButton from "$lib/components/settings/PickerRefreshButton.svelte";
  import QuickSaveStatus from "$lib/components/settings/QuickSaveStatus.svelte";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import {
    SYSTEM_DEFAULT_LABEL,
    SYSTEM_DEFAULT_MICROPHONE,
    microphoneChoiceFor,
    microphoneIDFor,
    microphoneLabel,
    microphoneMissing,
  } from "$lib/utils/microphone";

  let {
    editor,
    disabled = false,
  }: { editor: SettingsEditor; disabled?: boolean } = $props();
  const uid = $props.id();
  const choice = $derived(microphoneChoiceFor(editor.applied?.microphoneID));
  const label = $derived(microphoneLabel(choice, editor.devices));
  const missing = $derived(microphoneMissing(choice, editor.devices));
  const busy = $derived(
    disabled ||
      !editor.applied ||
      editor.dirty ||
      editor.saving ||
      editor.quickSettingsPending.length > 0 ||
      editor.applied.configuration.recoveryRequired,
  );

  function choose(next: string) {
    if (busy || !next || next === choice) return;
    void editor.updateQuickSettings(
      { microphoneID: microphoneIDFor(next) },
      "microphone",
    );
  }
</script>

<div class="min-w-0 space-y-1.5" role="group" aria-label="Microphone input">
  <div class="flex items-center justify-between gap-2">
    <label for={`${uid}-microphone`} class="content-value">Microphone</label>
    <PickerRefreshButton
      sidebar
      label="Refresh microphones"
      busy={editor.devicesBusy}
      disabled={busy}
      onclick={() => void editor.refreshDevices()}
    />
  </div>
  <Select.Root type="single" bind:value={() => choice, choose} disabled={busy}>
    <Select.Trigger
      id={`${uid}-microphone`}
      class="w-full"
      title={label}
      aria-describedby={missing ? `${uid}-microphone-missing` : undefined}
    >
      {label}
    </Select.Trigger>
    <Select.Content>
      <Select.Group>
        <Select.Item value={SYSTEM_DEFAULT_MICROPHONE}
          >{SYSTEM_DEFAULT_LABEL}</Select.Item
        >
        {#each editor.devices as device (device.id)}
          <Select.Item value={device.id}>{device.name}</Select.Item>
        {/each}
      </Select.Group>
    </Select.Content>
  </Select.Root>
  {#if missing}
    <p
      id={`${uid}-microphone-missing`}
      class="text-xs text-warning"
      role="status"
    >
      This microphone is disconnected. Reconnect it or choose another input;
      your saved choice is kept.
    </p>
  {/if}
  <QuickSaveStatus
    quiet
    fields={["microphone"]}
    pending={editor.quickSettingsPending}
    saved={editor.quickSettingsSaved}
    failed={editor.quickSettingsFailed}
  />
</div>
