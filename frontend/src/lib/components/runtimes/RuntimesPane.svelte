<script lang="ts">
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import RuntimeDetail from "./RuntimeDetail.svelte";
  import { Button } from "$lib/components/ui/button";
  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
  import type { Session } from "$lib/stores/session.svelte";

  let {
    session,
    onOpenConnections,
    workBusy = false,
  }: {
    session: Session;
    onOpenConnections: () => void;
    workBusy?: boolean;
  } = $props();

  const runtime = $derived(session.runtime);
  let selected = $state("");
  let recoveryID = $state("");

  const rows = $derived(
    runtime.providers.map((entry) => {
      const instance = runtime.instances.find(
        (item) => item.instance.provider === entry.id,
      );
      return {
        entry,
        instance,
        view: runtimePresentation(instance?.status),
      };
    }),
  );

  // Land on something real: a running runtime first, then anything installed.
  const active = $derived(
    rows.find((row) => row.entry.id === selected)?.entry.id ||
      rows.find((row) => row.instance?.status.state === "running")?.entry.id ||
      rows.find((row) => row.instance)?.entry.id ||
      rows[0]?.entry.id ||
      "",
  );
  // Retain that selection: stopping one provider must not move the user to
  // another provider that happens to still be running.
  $effect(() => {
    if (selected !== active) selected = active;
  });
  const current = $derived(rows.find((row) => row.entry.id === active));
  const providerRows = $derived(
    runtime.instances
      .filter((row) => row.instance.provider === active)
      .toSorted((a, b) => a.instance.id.localeCompare(b.instance.id)),
  );
  const currentInstance = $derived(
    providerRows.find((row) => row.instance.id === recoveryID) ??
      providerRows[0],
  );

  function tone(state: string | undefined): string {
    if (state === "running") return "bg-success";
    if (state === "error") return "bg-destructive";
    if (state === "starting" || state === "installing" || state === "stopping")
      return "bg-primary";
    if (state === "not_installed" || !state) return "bg-meter-rest";
    return "bg-muted-foreground";
  }
</script>

<!--
  The runtime inventory is a place of its own, not a section three clicks deep
  in configuration. The sidebar is the inventory and what this machine can
  actually run; the body is whichever runtime you picked.
-->
<div class="flex min-h-0 flex-1">
  <div
    class="flex w-[168px] shrink-0 flex-col overflow-y-auto border-r border-hairline bg-layer-fill min-[760px]:w-[252px]"
  >
    <SidebarHeader title="Runtimes">
      {#snippet actions()}
        <Button
          variant="ghost"
          size="xs"
          class="size-6 rounded-sm p-0"
          disabled={runtime.loading}
          aria-label="Refresh inventory"
          title="Refresh inventory"
          onclick={() => void runtime.load()}
          ><RefreshCwIcon class="size-3.5" /></Button
        >
      {/snippet}
    </SidebarHeader>

    <div
      class="flex flex-col gap-0.5 p-1.5"
      role="group"
      aria-label="Runtime inventory"
    >
      {#each rows as row (row.entry.id)}
        {@const on = row.entry.id === active}
        <button
          type="button"
          aria-pressed={on}
          class="flex items-center gap-2.5 rounded-md px-2 py-2 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {on
            ? 'bg-accent-wash'
            : ''}"
          onclick={() => {
            selected = row.entry.id;
            recoveryID = "";
          }}
        >
          <span
            class="flex size-6 shrink-0 items-center justify-center rounded-md border border-hairline bg-background"
          >
            <ProviderIcon profile={row.entry.id} size={14} />
          </span>
          <span class="min-w-0 flex-1">
            <span
              class="block truncate text-[13px] {on
                ? 'text-foreground'
                : 'text-secondary-foreground'}">{row.entry.name}</span
            >
            <span class="block truncate font-mono text-[10px] text-ink-quiet">
              {[
                row.entry.id === "llama-cpp"
                  ? "cleanup"
                  : row.instance?.status.realtime
                    ? "streaming"
                    : "speech",
                row.instance?.status.version || row.entry.version,
              ]
                .filter(Boolean)
                .join(" · ") || row.view.label}
            </span>
          </span>
          <span
            class="size-[7px] shrink-0 rounded-full {tone(
              row.instance?.status.state,
            )}"
            aria-hidden="true"
          ></span>
        </button>
      {/each}
      {#if !rows.length}
        <p class="px-2 py-3 text-xs text-muted-foreground">
          {runtime.loading
            ? "Reading the inventory…"
            : "No managed runtimes are available for this platform."}
        </p>
      {/if}
    </div>
    {#if runtime.error}
      <p class="px-3 py-3 text-[12px] text-destructive" role="alert">
        {runtime.error}
      </p>
    {/if}

    {#if current?.entry}
      <div class="mt-2 border-t border-hairline px-3 pt-3 pb-4">
        <p
          class="mb-2 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
        >
          This machine
        </p>
        <!-- Freehand knows which backends this machine can actually run;
               it does not measure VRAM, so this states capability rather
               than inventing a hardware meter. -->
        <p class="font-mono text-[10px] text-ink-quiet">
          {currentInstance?.status.backend
            ? backendLabel(currentInstance.status.backend)
            : "backend not selected"}
        </p>
        <div class="mt-2 flex flex-wrap gap-1">
          {#each current.entry.backends ?? [] as backend (backend)}
            <span
              class="rounded-sm border border-border px-1.5 py-0.5 text-[10px] text-secondary-foreground"
              >{backendLabel(backend)}</span
            >
          {/each}
        </div>
        {#if current.entry.unavailableReason}
          <p class="mt-2 text-[11px] leading-snug text-warning">
            {current.entry.unavailableReason}
          </p>
        {/if}
      </div>
    {/if}
  </div>

  {#if current}
    <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      {#if providerRows.length > 1}
        <div
          class="space-y-2 border-b border-warning/30 bg-warning/10 px-5 py-3"
        >
          <p class="text-[12px] text-secondary-foreground" role="status">
            Multiple saved installations need review. Reassign their Connections
            and remove unwanted files before deleting duplicate entries. Nothing
            is removed automatically.
          </p>
          <div class="flex flex-wrap gap-1.5">
            {#each providerRows as duplicate (duplicate.instance.id)}
              <Button
                variant="outline"
                size="xs"
                aria-pressed={currentInstance?.instance.id ===
                  duplicate.instance.id}
                onclick={() => (recoveryID = duplicate.instance.id)}
                >{duplicate.instance.name} · {duplicate.instance.id}</Button
              >
            {/each}
          </div>
        </div>
      {/if}
      {#key `${current.entry.id}/${currentInstance?.instance.id ?? ""}`}
        <RuntimeDetail
          {runtime}
          row={currentInstance}
          entry={current.entry}
          locked={workBusy || session.editor.saving || runtime.loading}
          {workBusy}
          {onOpenConnections}
        />
      {/key}
    </div>
  {:else}
    <div class="flex min-w-0 flex-1 items-center justify-center px-5">
      <p class="text-[13px] text-muted-foreground">
        No managed runtime is available for this platform.
      </p>
    </div>
  {/if}
</div>
