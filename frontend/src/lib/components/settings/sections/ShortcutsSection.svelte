<script lang="ts">
  import { platformPresentation } from "$lib/platform";
  import { getContext, onMount } from "svelte";
  import * as SettingsService from "$bindings/settings/service";
  import * as InputService from "$bindings/input/service";
  import {
    NativePermissionState,
    NATIVE_PERMISSION_SERVICES,
    type NativePermissionServices,
  } from "$lib/stores/nativePermissions.svelte";
  const permissions = new NativePermissionState(
    getContext<NativePermissionServices | undefined>(
      NATIVE_PERMISSION_SERVICES,
    ) ?? InputService,
  );
  const retryNative =
    getContext<(() => Promise<Settings>) | undefined>("hold-shortcut-retry") ??
    SettingsService.RetryHoldShortcut;
  let retrying = $state(false);
  let retryError = $state("");
  let retryMessage = $state("");
  let disposed = false;
  async function retryHold() {
    if (retrying || !editable || capture.capturing) return;
    retrying = true;
    retryError = "";
    retryMessage = "";
    try {
      const latest = await retryNative();
      if (disposed) return;
      // Never replace the user's unsaved draft with the saved retry snapshot.
      settings.holdAvailable = latest.holdAvailable;
      settings.holdAvailabilityReason = latest.holdAvailabilityReason;
      retryMessage = latest.holdAvailabilityReason;
    } catch (cause) {
      if (disposed) return;
      retryError = String(cause).replace(/^(?:Runtime)?Error:\s*/, "");
      // Wails rejects a (snapshot, error) response without delivering the DTO.
      // A dirty editor also defers settings events, so explicitly refresh only
      // runtime availability without replacing the unsaved shortcut draft.
      try {
        const latest = await SettingsService.GetSettings();
        if (!disposed) {
          settings.holdAvailable = latest.holdAvailable;
          settings.holdAvailabilityReason = latest.holdAvailabilityReason;
        }
      } catch (refreshCause) {
        if (!disposed)
          retryError += ` Availability refresh failed: ${String(refreshCause).replace(/^(?:Runtime)?Error:\s*/, "")}`;
      }
    } finally {
      if (!disposed) {
        retrying = false;
        void permissions.refresh();
      }
    }
  }
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
    if (settings.platform === "darwin") void permissions.refresh();
    return () => {
      disposed = true;
      permissions.dispose();
    };
  });
  const native = $derived(platformPresentation(settings.platform));
</script>

<svelte:window
  onfocus={() => {
    if (native.mac) void permissions.refresh();
  }}
/>
<div class="flex flex-col gap-3">
  {#if capture.policyError}
    <div
      class="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/8 px-4 py-3"
      role="alert"
    >
      <p class="text-[13px] text-destructive">{capture.policyError}</p>
      <Button variant="outline" size="sm" onclick={() => capture.loadPolicies()}
        >Retry</Button
      >
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
            platform={settings.platform}
            id={`${policy.action}-shortcut`}
            title={shortcutLabel(policy.action)}
            description={shortcutDescription(policy.action)}
            requirement={shortcutRequirement(policy)}
            value={shortcutValue(settings, policy.action)}
            preview={capture.active === policy.action ? capture.preview : ""}
            capturing={capture.active === policy.action}
            disabled={!editable ||
              retrying ||
              capture.busyElsewhere(policy.action)}
            captureUnavailable={unavailable(policy.action) ||
              (native.mac &&
                (!permissions.status?.keyboard ||
                  !permissions.status?.accessibility))}
            clearable={!policy.required}
            restorable={Boolean(policy.defaultShortcut) &&
              !isRecommendedShortcut(
                policy,
                shortcutValue(settings, policy.action),
              )}
            feedback={capture.feedbackFor(policy.action)}
            onRecord={() => capture.record(settings, policy.action)}
            onCancel={() => capture.cancel()}
            onClear={!policy.required
              ? () => capture.clear(settings, policy.action)
              : undefined}
            onRestore={() => capture.restore(settings, policy)}
          />
        {/each}
      {/if}
    </SettingsCard>
  {/if}

  {#if !settings.holdAvailable}
    <p class="section-footnote">
      {settings.holdAvailabilityReason}
    </p>
  {/if}

  {#if native.mac}
    <Button
      variant="soft"
      class="self-start"
      size="sm"
      disabled={!editable || retrying || capture.capturing}
      onclick={retryHold}
    >
      {retrying ? "Retrying hold-to-talk…" : "Retry hold-to-talk"}
    </Button>
    {#if retryError}<p role="alert" class="px-1 text-sm text-destructive">
        {retryError}
      </p>{/if}
    {#if retryMessage}<p role="status" class="px-1 text-sm">
        {retryMessage}
      </p>{/if}
    {#if permissions.error}<p
        role="alert"
        class="px-1 text-sm text-destructive"
      >
        {permissions.error}
      </p>{/if}
    <p class="section-footnote">
      Toggle and Show Freehand use native global shortcuts without
      Accessibility. Recording a shortcut requires Accessibility and Input
      Monitoring; clearing an optional shortcut does not. Hold-to-talk requires
      Input Monitoring. After unlocking or leaving Secure Input, release all
      keys and click Retry hold-to-talk to rearm the saved shortcut. Save any
      shortcut edits first.
    </p>
  {/if}

  {#if externalAvailabilityDeferred}
    <p class="section-footnote">
      {#if native.mac}
        Freehand rejects unsupported, reserved, or duplicate chords. macOS
        shortcuts used by other applications may still conflict. Review
        permission access in General settings if native shortcut capture is
        unavailable. Choose another chord if macOS intercepts it.
      {:else}
        Freehand can reject unsupported, reserved, or duplicate chords
        immediately. Windows can only reveal a shortcut already owned by another
        application when you save; if registration fails, your previous working
        shortcuts stay active. On keyboard layouts where Ctrl+Alt acts as AltGr,
        choose a different chord.
      {/if}
    </p>
  {/if}
</div>
