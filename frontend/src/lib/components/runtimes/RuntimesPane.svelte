<script lang="ts">
  import { onDestroy } from "svelte";
  import type { WorkflowPane } from "$lib/panes";
  import {
    runtimeWorkflowConnection,
    runtimeWorkflowTargets,
    runtimeUseBlockReason,
  } from "$lib/utils/runtimeWorkflowUse";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import RuntimeStatus from "$lib/components/common/RuntimeStatus.svelte";
  import CpuIcon from "@lucide/svelte/icons/cpu";
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import SidebarContribution from "$lib/components/shell/SidebarContribution.svelte";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import RuntimeDetail from "./RuntimeDetail.svelte";
  import { Button } from "$lib/components/ui/button";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { runtimeSidebarContext } from "$lib/utils/runtimeSidebar";
  import type { Session } from "$lib/stores/session.svelte";

  let {
    session,
    onOpenConnections,
    workflow = "voice",
    onUseRuntime,
    onReturnToWorkflow,
    workBusy = false,
  }: {
    session: Session;
    onOpenConnections: () => void;
    workflow?: WorkflowPane;
    onUseRuntime?: (instanceID: string) => Promise<boolean>;
    onReturnToWorkflow?: () => void;
    workBusy?: boolean;
  } = $props();
  const workbench = getWorkbenchLayout();
  const uid = $props.id();

  const runtime = $derived(session.runtime);
  let selected = $state("");
  let recoveryID = $state("");

  const rows = $derived(
    runtime.providers.map((entry) => {
      const installations = runtime.instances
        .filter((item) => item.instance.provider === entry.id)
        .toSorted((a, b) => a.instance.id.localeCompare(b.instance.id));
      const instance =
        installations.find((item) => item.instance.id === recoveryID) ??
        installations[0];
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
        problem: runtime.errorFor(id) || view.error,
        context: runtimeSidebarContext(entry, instance),
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
        // A provider hint accompanying an exact installation is part of the
        // same request, not a second request to select the first installation.
        workbench.runtimeProviderID = "";
        selectionGeneration++;
        selected = requested.instance.provider;
        recoveryID = requestedID;
        return;
      }
    }
    const requestedProvider = workbench?.runtimeProviderID;
    if (requestedProvider) {
      if (runtime.loading) return;
      workbench.runtimeProviderID = "";
      if (rows.some((row) => row.entry.id === requestedProvider)) {
        selectionGeneration++;
        selected = requestedProvider;
        recoveryID = "";
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
  const currentContext = $derived(
    current
      ? runtimeSidebarContext(
          current.entry,
          currentInstance,
          runtime.pendingFor(currentInstance?.instance.id ?? current.entry.id),
          runtime.errorFor(currentInstance?.instance.id ?? current.entry.id),
        )
      : null,
  );
  const workflowTarget = $derived(runtimeWorkflowTargets[workflow]);
  const workflowConnection = $derived(
    currentInstance
      ? runtimeWorkflowConnection(
          session,
          currentInstance.instance.id,
          workflow,
        )
      : undefined,
  );
  const workflowSelected = $derived(
    !!workflowConnection &&
      session.editor.applied?.savedConnections.selected?.[
        workflowTarget.purpose
      ] === workflowConnection.id,
  );
  const useBlockReason = $derived(runtimeUseBlockReason(session, workBusy));
  let usePending = $state(false);
  let useError = $state("");
  let alive = true;
  let selectionGeneration = 0;
  const selectionKey = $derived(
    `${workflow}/${currentInstance?.instance.id ?? ""}`,
  );
  $effect(() => {
    selectionKey;
    selectionGeneration++;
    useError = "";
  });
  onDestroy(() => {
    alive = false;
  });
  async function useForWorkflow() {
    if (
      !currentInstance ||
      !workflowConnection ||
      !onUseRuntime ||
      !onReturnToWorkflow ||
      usePending ||
      useBlockReason
    )
      return;
    const id = currentInstance.instance.id;
    const generation = selectionGeneration;
    const origin = workflow;
    useError = "";
    usePending = true;
    try {
      const success = await onUseRuntime(id);
      // A newer provider choice or leaving Runtimes must not be undone by an
      // older save response. Selection itself stays owned by SettingsEditor.
      if (
        !alive ||
        generation !== selectionGeneration ||
        workflow !== origin ||
        currentInstance?.instance.id !== id
      )
        return;
      if (success) onReturnToWorkflow();
      else
        useError = `Could not select this runtime for ${workflowTarget.label}. Review the error and try again.`;
    } catch {
      if (alive && generation === selectionGeneration)
        useError = `Could not select this runtime for ${workflowTarget.label}. Try again.`;
    } finally {
      if (alive) usePending = false;
    }
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
              selectionGeneration++;
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
              <span
                class="block break-words text-xs text-ink-quiet [overflow-wrap:anywhere]"
                title={`${row.context.model.label}: ${row.context.model.name}`}
              >
                {row.context.model.label === "Recommended model"
                  ? "Recommended: "
                  : ""}{row.context.model.name}
              </span>
              <span class="mt-0.5 block">
                <RuntimeStatus
                  view={row.view}
                  problem={row.problem}
                  label={row.instance
                    ? row.view.label
                    : row.entry.supported
                      ? "Not installed"
                      : "Unavailable"}
                />
              </span>
            </span>
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

      {#if currentContext}
        <section
          class="mt-2 space-y-3 border-t border-hairline px-3 pt-3 pb-4"
          aria-label="Runtime setup summary"
        >
          <div>
            <p class="content-kicker mb-1">{currentContext.model.label}</p>
            <p
              class="break-words text-[13px] font-medium text-foreground [overflow-wrap:anywhere]"
              title={currentContext.model.id}
            >
              {currentContext.model.name}
            </p>
            {#if currentContext.speechModel}
              <p class="mt-2 text-xs text-muted-foreground">
                Selected speech model
              </p>
              <p
                class="mt-0.5 break-words text-xs text-secondary-foreground [overflow-wrap:anywhere]"
                title={currentContext.speechModel.id}
              >
                {currentContext.speechModel.name}
              </p>
            {/if}
          </div>
          <div>
            <p class="content-kicker mb-1">{currentContext.heading}</p>
            <p class="text-[13px] font-medium text-foreground">
              {currentContext.next}
            </p>
            {#if currentContext.detail}
              <p
                class="mt-1 break-words text-xs leading-relaxed text-muted-foreground [overflow-wrap:anywhere]"
              >
                {currentContext.detail}
              </p>
            {/if}
          </div>
          {#if currentContext.binary}
            <p
              class="flex items-start gap-1.5 text-xs text-ink-quiet"
              title="Installed runtime binary"
            >
              <CpuIcon class="mt-px size-3 shrink-0" aria-hidden="true" />
              <span class="min-w-0 break-words">{currentContext.binary}</span>
            </p>
          {/if}
        </section>
      {/if}
    </div>
  </SidebarContribution>

  {#if current}
    <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      {#snippet useAction()}
        {#if workflowConnection && onUseRuntime && onReturnToWorkflow}
          <Button
            size="xs"
            disabled={usePending || !!useBlockReason}
            title={useBlockReason || undefined}
            onclick={() => void useForWorkflow()}
            >{usePending
              ? "Selecting…"
              : workflowSelected
                ? `Open ${workflowTarget.label}`
                : `Use for ${workflowTarget.label}`}</Button
          >
        {/if}
      {/snippet}
      {#snippet recoveryNotice()}
        {#if useError}<p
            class="border-b border-hairline px-3 py-2 text-xs text-destructive"
            role="alert"
          >
            {useError}
          </p>{/if}
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
                    selectionGeneration++;
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
          connectionCatalog={session.editor.applied?.savedConnections}
          notice={recoveryNotice}
          {useAction}
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
