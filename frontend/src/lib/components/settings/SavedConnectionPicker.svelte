<script lang="ts">
  import type { InstanceStatus } from "$bindings/managedruntime";
  import {
    type Catalog,
    type Change,
    type Purpose,
  } from "$bindings/savedconnection";
  import { connectionTargetLabel } from "$lib/utils/connectionChoices";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button } from "$lib/components/ui/button";
  let {
    catalog,
    runtimeInstances,
    purpose,
    dirty,
    busy,
    inactive = false,
    onChange,
    onManage,
    onBrowse,
    onAdd,
  }: {
    catalog: Catalog;
    runtimeInstances: InstanceStatus[];
    purpose: Purpose;
    dirty: boolean;
    busy: boolean;
    inactive?: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onManage: () => void;
    onBrowse: () => void;
    onAdd: () => void;
  } = $props();
  const entries = $derived(
    (catalog.entries ?? []).filter(
      (c) => c.uses?.includes(purpose) || c.id === catalog.selected?.[purpose],
    ),
  );
  const selected = $derived(
    entries.find((c) => c.id === catalog.selected?.[purpose]),
  );
</script>

<section
  class="border-t border-hairline py-3"
  aria-label={inactive ? "Saved manual connection" : "Active connection"}
>
  <div class="flex min-w-0 flex-col gap-1.5">
    <label for={`saved-connection-${purpose}`} class="content-value"
      >Connection</label
    >
    <div class="min-w-0">
      <ConnectionSelect
        id={`saved-connection-${purpose}`}
        {catalog}
        {runtimeInstances}
        {purpose}
        disabled={busy}
        {onChange}
        {onAdd}
        onManage={onBrowse}
      />
    </div>
  </div>
  <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-2">
    <Button
      variant="outline"
      size="xs"
      disabled={busy}
      onclick={selected ? onManage : onAdd}
      >{selected?.builtIn
        ? "Connection details"
        : selected
          ? "Edit connection"
          : "Add connection"}</Button
    >
    {#if selected}<p
        class="min-w-0 break-words text-xs text-muted-foreground [overflow-wrap:anywhere]"
        title={connectionTargetLabel(selected, runtimeInstances)}
      >
        {connectionTargetLabel(selected, runtimeInstances)}
      </p>
    {:else}<p class="mt-2 text-xs text-muted-foreground">
        {entries.length
          ? "Choose a saved connection to configure this feature."
          : "Add a server connection or set up a local runtime. Built-in connections appear automatically."}
      </p>{/if}
  </div>
  {#if dirty}<p class="mt-2 text-xs text-muted-foreground">
      Save or discard your edits before switching connections.
    </p>{/if}
</section>
