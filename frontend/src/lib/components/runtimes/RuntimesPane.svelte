<script lang="ts">
  import PlusIcon from "@lucide/svelte/icons/plus";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import LocalRuntimeSection from "$lib/components/settings/sections/LocalRuntimeSection.svelte";
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
    selected ||
      rows.find((row) => row.instance?.status.state === "running")?.entry.id ||
      rows.find((row) => row.instance)?.entry.id ||
      rows[0]?.entry.id ||
      "",
  );
  const current = $derived(rows.find((row) => row.entry.id === active));

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
<div class="flex min-h-0 flex-1 flex-col">
  <PaneHeader
    title="Local runtime"
    summary="installed engines and their models"
  >
    {#snippet actions()}
      <Button
        variant="outline"
        size="xs"
        disabled={runtime.loading}
        onclick={() => void runtime.load()}
      >
        <RefreshCwIcon class="size-3" />
        Refresh inventory
      </Button>
      <Button variant="outline" size="xs" onclick={onOpenConnections}>
        Connections
      </Button>
    {/snippet}
  </PaneHeader>

  <div class="flex min-h-0 flex-1">
    <div
      class="flex w-[252px] shrink-0 flex-col overflow-y-auto border-r border-hairline bg-layer-fill"
    >
      <div
        class="flex h-8 shrink-0 items-center justify-between border-b border-hairline px-3"
      >
        <span
          class="text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
          >Runtimes</span
        >
        <PlusIcon class="size-3.5 text-muted-foreground" aria-hidden="true" />
      </div>

      <div
        class="flex flex-col gap-0.5 p-1.5"
        role="listbox"
        aria-label="Installed runtimes"
      >
        {#each rows as row (row.entry.id)}
          {@const on = row.entry.id === active}
          <button
            type="button"
            role="option"
            aria-selected={on}
            class="flex items-center gap-2.5 rounded-md px-2 py-2 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {on
              ? 'bg-accent-wash'
              : ''}"
            onclick={() => (selected = row.entry.id)}
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
                  : 'text-secondary-foreground'}">{row.entry.id}</span
              >
              <span class="block truncate font-mono text-[10px] text-ink-quiet">
                {row.view.label}{row.instance?.status.backend
                  ? ` · ${backendLabel(row.instance.status.backend)}`
                  : ""}
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
            {current.instance?.status.backend
              ? backendLabel(current.instance.status.backend)
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

    <div class="min-w-0 flex-1 overflow-y-auto px-5 py-4">
      <LocalRuntimeSection
        {runtime}
        focus={active}
        chrome={false}
        {workBusy}
        disabled={session.editor.saving}
        onAction={(action) => action()}
        onConnections={onOpenConnections}
      />
    </div>
  </div>
</div>
