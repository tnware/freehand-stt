<script lang="ts">
  import { getContext, onMount } from "svelte";
  import * as InputService from "$bindings/input/service";
  import {
    NativePermissionState,
    NATIVE_PERMISSION_SERVICES,
    type NativePermissionServices,
  } from "$lib/stores/nativePermissions.svelte";
  import { permissionRows } from "$lib/utils/nativePermissions";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingRow from "./SettingRow.svelte";
  import { Button } from "$lib/components/ui/button";
  const permissions = new NativePermissionState(
    getContext<NativePermissionServices | undefined>(
      NATIVE_PERMISSION_SERVICES,
    ) ?? InputService,
  );
  const rows = $derived(permissionRows(permissions.status));
  onMount(() => {
    void permissions.refresh();
    return () => permissions.dispose();
  });
</script>

<svelte:window onfocus={() => permissions.refresh()} />
<SettingsCard>
  <SettingRow title="macOS permissions">
    {#snippet control()}
      <Button
        variant="outline"
        size="sm"
        disabled={permissions.busy}
        onclick={() => permissions.refresh()}>Refresh permissions</Button
      >
    {/snippet}
    <p class="text-xs text-muted-foreground">
      Audio files and text-to-speech do not require these permissions.
    </p>
    {#if permissions.error}<p
        role="alert"
        class="mt-3 text-sm text-destructive"
      >
        {permissions.error}
      </p>{/if}
    {#if permissions.busy}<p
        role="status"
        class="mt-3 text-xs text-muted-foreground"
      >
        Checking permissions…
      </p>{/if}
    {#each rows as row (row.kind)}
      <SettingRow title={row.label} description={row.description} compact>
        {#snippet control()}
          <span class="text-xs text-muted-foreground">{row.state}</span>
          {#if row.action}
            <Button
              variant="outline"
              size="sm"
              disabled={permissions.busy}
              onclick={() =>
                row.action === "request"
                  ? permissions.request(row.kind)
                  : permissions.openSettings(row.kind)}
              >{row.action === "request"
                ? `Allow ${row.label}`
                : `Open ${row.label} settings`}</Button
            >
          {/if}
        {/snippet}
      </SettingRow>
    {/each}
    <p class="mt-2 text-xs text-muted-foreground">
      Toggle and Show Freehand shortcuts do not require Accessibility or Input
      Monitoring.
    </p>
    <p class="mt-2 text-xs text-muted-foreground">
      After changing access, return here to refresh. macOS may require you to
      quit and reopen Freehand.
    </p>
  </SettingRow>
</SettingsCard>
