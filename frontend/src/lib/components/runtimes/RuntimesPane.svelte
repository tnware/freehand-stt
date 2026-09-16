<script lang="ts">
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import CpuIcon from "@lucide/svelte/icons/cpu";
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import SidebarContribution from "$lib/components/shell/SidebarContribution.svelte";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
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
  const workbench = getWorkbenchLayout();
  const uid = $props.id();

  const runtime = $derived(session.runtime);
  let selected = $state("");
  let recoveryID = $state("");

  const rows = $derived(
    runtime.providers.map((entry) => {
      const instance = runtime.instances.find(
        (item) => item.instance.provider === entry.id,
      );
      const id = instance?.instance.id ?? entry.id;
      const view = runtimePresentation(
        instance?.status,
        undefined,
        runtime.pendingFor(id),
      );
      return {
        entry,
        instance,
        view,
        operating: runtime.isBusy(id),
        problem: runtime.errorFor(id) || view.error,
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
    const requestedID = workbench?.runtimeInstanceID;
    if (requestedID) {
      if (runtime.loading) return;
      const requested = runtime.statusFor(requestedID);
      workbench.runtimeInstanceID = "";
      if (
        requested &&
        rows.some((row) => row.entry.id === requested.instance.provider)
      ) {
        selected = requested.instance.provider;
        recoveryID = requestedID;
        return;
      }
    }
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
<div class="flex min-h-0 min-w-0 flex-1">
  <SidebarContribution id="runtimes">
    <div
      class="flex min-h-0 shrink-0 flex-col overflow-y-auto border-r border-hairline bg-layer-fill {workbench
        ? 'h-full w-full'
        : 'w-[168px] min-[760px]:w-[252px]'}"
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
            ><RefreshCwIcon
              class="size-3.5 {runtime.loading ? 'animate-spin' : ''}"
            /></Button
          >
        {/snippet}
      </SidebarHeader>

      <div
        class="flex flex-col py-1.5"
        role="group"
        aria-label="Runtime inventory"
      >
        {#each rows as row (row.entry.id)}
          {@const on = row.entry.id === active}
          <button
            type="button"
            aria-pressed={on}
            aria-describedby={row.problem
              ? `${uid}-${row.entry.id}-error`
              : undefined}
            class="flex min-h-11 items-center gap-2.5 border-l-2 pl-2.5 pr-3 py-1.5 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {on
              ? 'border-primary bg-accent-wash-strong'
              : 'border-transparent'}"
            onclick={() => {
              selected = row.entry.id;
              recoveryID = "";
              if (workbench?.compact.current && workbench.compactPrimaryOpen)
                workbench.closePrimary();
            }}
          >
            <span
              class="flex size-6 shrink-0 items-center justify-center rounded-sm bg-subtle-fill-hover"
            >
              <ProviderIcon profile={row.entry.id} size={14} />
            </span>
            <span class="min-w-0 flex-1">
              <span
                class="block truncate text-[13px] {on
                  ? 'text-foreground'
                  : 'text-secondary-foreground'}">{row.entry.name}</span
              >
              <span class="block truncate font-mono text-xs text-ink-quiet">
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
              <span
                class="mt-0.5 flex items-start gap-1.5 text-xs leading-relaxed text-secondary-foreground"
              >
                {#if row.operating}<LoaderCircleIcon
                    class="mt-0.5 size-3 shrink-0 animate-spin"
                    aria-hidden="true"
                  />{/if}
                <span class="min-w-0 break-words"
                  >{row.instance
                    ? row.view.label
                    : row.entry.supported
                      ? "Not installed"
                      : "Unavailable"}</span
                >
              </span>
            </span>
            <span
              class="size-[7px] shrink-0 rounded-full {row.operating
                ? 'bg-primary'
                : row.problem
                  ? 'bg-destructive'
                  : tone(row.instance?.status.state)}"
              aria-hidden="true"
            ></span>
          </button>
          {#if row.problem}<p
              id={`${uid}-${row.entry.id}-error`}
              class="break-words px-2.5 pb-2 text-xs leading-relaxed text-destructive"
            >
              {row.problem}
            </p>{/if}
        {/each}
        {#if !rows.length}
          <p class="px-3 py-3 text-xs text-muted-foreground" role="status">
            {runtime.loading
              ? "Reading the inventory…"
              : "No managed runtimes are available for this platform."}
          </p>
        {/if}
      </div>
      {#if runtime.error}
        <p
          class="break-words px-3 py-3 text-[12px] text-destructive"
          role="alert"
        >
          {runtime.error}
        </p>
      {/if}

      {#if current?.entry}
        <div class="mt-2 border-t border-hairline px-3 pt-3 pb-4">
          <p class="content-kicker mb-2">This machine</p>
          <!-- Freehand knows which backends this machine can actually run;
               it does not measure VRAM, so this states capability rather
               than inventing a hardware meter. -->
          <p class="font-mono text-xs text-ink-quiet">
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
            <p class="mt-2 text-xs leading-snug text-warning">
              {current.entry.unavailableReason}
            </p>
          {/if}
        </div>
      {/if}
    </div>
  </SidebarContribution>

  {#if current}
    <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      {#snippet recoveryNotice()}
        {#if providerRows.length > 1}
          <div
            class="space-y-2 border-b border-warning/30 bg-warning/10 px-3 py-3"
          >
            <p class="text-[12px] text-secondary-foreground" role="status">
              Multiple saved installations need review. Reassign their
              Connections and remove unwanted files before deleting duplicate
              entries. Nothing is removed automatically.
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each providerRows as duplicate (duplicate.instance.id)}
                <Button
                  variant="outline"
                  size="xs"
                  aria-pressed={currentInstance?.instance.id ===
                    duplicate.instance.id}
                  onclick={() => {
                    recoveryID = duplicate.instance.id;
                    if (
                      workbench?.compact.current &&
                      workbench.compactPrimaryOpen
                    )
                      workbench.closePrimary();
                  }}>{duplicate.instance.name} · {duplicate.instance.id}</Button
                >
              {/each}
            </div>
          </div>
        {/if}
      {/snippet}
      {#key `${current.entry.id}/${currentInstance?.instance.id ?? ""}`}
        <RuntimeDetail
          {runtime}
          row={currentInstance}
          entry={current.entry}
          locked={workBusy || session.editor.saving || runtime.loading}
          {workBusy}
          {onOpenConnections}
          notice={recoveryNotice}
        />
      {/key}
    </div>
  {:else}
    <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      <PaneHeader title="Runtimes" icon={CpuIcon} />
      <div
        class="flex min-h-0 min-w-0 flex-1 items-center justify-center overflow-y-auto p-5"
      >
        <div class="max-w-sm space-y-3">
          <p class="content-section-title" role="status">
            {runtime.loading
              ? "Reading runtime inventory…"
              : runtime.error
                ? "Runtime inventory unavailable"
                : "No managed runtimes available"}
          </p>
          <p class="content-meta">
            {runtime.loading
              ? "Checking installations and supported runtimes on this machine."
              : runtime.error
                ? "Refresh the inventory to try again."
                : "This platform has no managed runtimes. Configure your speech server in Connections."}
          </p>
          {#if !runtime.loading}
            <Button
              variant="outline"
              size="xs"
              onclick={runtime.error
                ? () => void runtime.load()
                : onOpenConnections}
            >
              {runtime.error ? "Refresh inventory" : "Open Connections"}
            </Button>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>
