<script lang="ts">
  import { onMount } from "svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ShortcutRecorder from "$lib/components/settings/ShortcutRecorder.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { Settings, Status } from "$lib/state";
  import type { ShortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import {
    ShortcutAction,
    isRecommendedShortcut,
    shortcutDescription,
    shortcutEditingAllowed,
    shortcutLabel,
    shortcutRequirement,
    shortcutValue,
  } from "$lib/utils/shortcuts";

  let {
    settings = $bindable(),
    status,
    busy = false,
    capture,
  }: {
    settings: Settings;
    status: Status;
    busy?: boolean;
    capture: ShortcutCapture;
  } = $props();

  const editable = $derived(shortcutEditingAllowed(status, busy));
  const externalAvailabilityDeferred = $derived(
    capture.policies.some((policy) => !policy.externalAvailabilityKnown),
  );

  const unavailable = (action: ShortcutAction) =>
    action === ShortcutAction.HoldToTalk && !settings.holdAvailable;

  onMount(() => {
    void capture.loadPolicies();
  });
</script>

<div class="flex flex-col gap-4">
  {#if capture.policyError}
    <div
      class="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/8 px-4 py-3"
      role="alert"
    >
      <p class="text-[13px] text-destructive">{capture.policyError}</p>
      <Button variant="outline" size="sm" onclick={() => capture.loadPolicies()}>Retry</Button>
    </div>
  {:else}
    <SettingsCard>
      {#if capture.policiesLoading}
        <div class="px-5 py-6 text-sm text-muted-foreground" role="status">
          Loading native shortcut policy…
        </div>
      {:else}
        {#each capture.policies as policy (policy.action)}
          <ShortcutRecorder
            id={`${policy.action}-shortcut`}
            title={shortcutLabel(policy.action)}
            description={shortcutDescription(policy.action)}
            requirement={shortcutRequirement(policy)}
            value={shortcutValue(settings, policy.action)}
            preview={capture.active === policy.action ? capture.preview : ""}
            capturing={capture.active === policy.action}
            disabled={unavailable(policy.action) ||
              !editable ||
              capture.busyElsewhere(policy.action)}
            clearable={policy.action === ShortcutAction.HoldToTalk}
            restorable={Boolean(policy.defaultShortcut) &&
              !isRecommendedShortcut(policy, shortcutValue(settings, policy.action))}
            feedback={capture.feedbackFor(policy.action)}
            onRecord={() => capture.record(settings, policy.action)}
            onCancel={() => capture.cancel()}
            onClear={policy.action === ShortcutAction.HoldToTalk
              ? () => capture.clear(settings, policy.action)
              : undefined}
            onRestore={() => capture.restore(settings, policy)}
          />
        {/each}
      {/if}
    </SettingsCard>
  {/if}

  {#if !settings.holdAvailable}
    <p class="px-1 text-[13px] leading-relaxed text-muted-foreground">
      {settings.holdAvailabilityReason}
    </p>
  {/if}

  {#if externalAvailabilityDeferred}
    <p class="px-1 text-[13px] leading-relaxed text-muted-foreground">
      Freehand can reject unsupported, reserved, or duplicate chords immediately. Windows can only
      reveal a shortcut already owned by another application when you save; if registration fails,
      your previous working shortcuts stay active. On keyboard layouts where Ctrl+Alt acts as AltGr,
      choose a different chord.
    </p>
  {/if}
</div>
