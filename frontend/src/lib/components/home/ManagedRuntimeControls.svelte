<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import CpuIcon from "@lucide/svelte/icons/cpu";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let {
    runtime,
    instanceID,
    disabled = false,
    workBusy = false,
    sidebar = false,
    onManage,
  }: {
    runtime: ManagedRuntimeState;
    instanceID: string;
    disabled?: boolean;
    workBusy?: boolean;
    sidebar?: boolean;
    onManage: () => void;
  } = $props();
  const uid = $props.id();
  const layout = getWorkbenchLayout();
  let now = $state(Date.now());
  $effect(() => {
    if (status?.state !== "starting") return;
    const timer = setInterval(() => (now = Date.now()), 1000);
    return () => clearInterval(timer);
  });
  const row = $derived(runtime.statusFor(instanceID));
  const status = $derived(row?.status);
  const pending = $derived(runtime.pendingFor(instanceID));
  const view = $derived(runtimePresentation(status, now, pending));
  const provider = $derived(
    runtime.providers.find((p) => p.id === row?.instance.provider),
  );
  const operating = $derived(runtime.isBusy(instanceID));
  const running = $derived(status?.state === "running");
  const locked = $derived(
    disabled || workBusy || operating || runtime.loading || !status?.supported,
  );
  const models = $derived(
    (status?.models ?? []).filter(
      (m) => m.installed && provider?.models?.some((q) => q.id === m.id),
    ),
  );
  const selected = $derived(
    status?.models?.find((m) => m.id === row?.instance.model),
  );
  const choices = $derived(models.map((m) => ({ value: m.id, label: m.name })));
  const problem = $derived(
    runtime.errorFor(instanceID) || view.error || (!row ? runtime.error : ""),
  );
  const label = $derived(
    !row && runtime.loading ? "Checking runtime" : view.label,
  );
  const needsSetup = $derived(
    !!row && !operating && (!view.installed || !selected?.installed),
  );
  const modelLocked = $derived(locked || running || !models.length);

  function manage() {
    if (layout) layout.runtimeInstanceID = instanceID;
    onManage();
  }
  function output() {
    if (layout) {
      if (!layout.bottomAvailable.current) return;
      layout.showOutput(instanceID);
      layout.closePrimary();
    } else void runtime.openOutput(instanceID);
  }
  function chooseModel(model: string) {
    if (
      !row ||
      modelLocked ||
      model === row.instance.model ||
      !models.some((m) => m.id === model)
    )
      return;
    void runtime.saveInstance({ ...row.instance, model });
  }
  function command() {
    if (locked || (!running && (!view.installed || !selected?.installed)))
      return;
    void runtime.run(instanceID, running ? "Stop" : "Start");
  }
  function cancel() {
    if (!disabled && operating && pending !== "Cancelling")
      void runtime.cancel(instanceID);
  }
</script>

<div
  class={sidebar ? "flex min-w-0 flex-col gap-1.5" : "min-w-0 space-y-2.5"}
  class:runtime-sidebar={sidebar}
  role="group"
  aria-label={`Local runtime ${row?.instance.name || instanceID}`}
>
  <div class="flex items-center justify-between gap-2">
    {#if sidebar}
      <div class="flex min-w-0 items-start gap-1.5">
        <span
          class="text-[11px] leading-relaxed text-muted-foreground"
          title="Local runtime">Local</span
        >
        {@render runtimeStatus()}
      </div>
    {:else}<span
        class="flex min-w-0 items-center gap-1.5 text-[11px] font-medium text-muted-foreground"
      >
        <CpuIcon class="size-3.5 shrink-0" aria-hidden="true" />Local runtime
      </span>{/if}
    {#if operating}
      <Button
        variant="outline"
        size="xs"
        disabled={disabled || pending === "Cancelling"}
        onclick={cancel}>Cancel</Button
      >
    {:else if needsSetup}
      <Button variant="soft" size="xs" onclick={manage}>Set up runtime</Button>
    {:else}
      <Button
        variant={running ? "outline" : "soft"}
        size="xs"
        disabled={locked ||
          (!running && (!view.installed || !selected?.installed))}
        title={workBusy
          ? "Finish the current task before changing this runtime."
          : undefined}
        onclick={command}
      >
        {#if running}<SquareIcon class="size-3" />Stop{:else}<PlayIcon
            class="size-3"
          />Start{/if}
      </Button>
    {/if}
  </div>
  {#if !sidebar}{@render runtimeStatus()}{/if}
  {#if view.startup}<p
      class="text-[11px] leading-relaxed text-muted-foreground"
    >
      {view.startup}
    </p>{/if}
  {#if view.transferred}<p class="font-mono text-[11px] text-muted-foreground">
      {view.transferred}
    </p>{/if}
  {#if !operating && view.completion && !problem && status?.operation?.outcome === "cancelled"}<p
      class="text-xs text-muted-foreground"
    >
      {view.completion}
    </p>{/if}
  {#if problem}<p
      class="break-words text-xs leading-relaxed text-destructive"
      role="alert"
    >
      {problem}
    </p>{/if}

  {#if row}
    <div class={sidebar ? "min-w-0" : "space-y-1.5"}>
      <label
        for={`${uid}-model`}
        class={sidebar ? "sr-only" : "text-xs font-medium"}
        >Selected model</label
      >
      <Select.Root
        type="single"
        bind:value={() => row.instance.model, chooseModel}
        items={choices}
        disabled={modelLocked}
      >
        <Select.Trigger
          id={`${uid}-model`}
          class="h-7 w-full min-w-0 text-xs"
          title={running
            ? "Stop the runtime to select another downloaded model."
            : undefined}
        >
          <span class="min-w-0 truncate"
            >{selected?.name || row.instance.model || "Choose a model"}</span
          >
        </Select.Trigger>
        <Select.Content
          >{#each models as model (model.id)}<Select.Item
              value={model.id}
              label={model.name}>{model.name}</Select.Item
            >{/each}</Select.Content
        >
      </Select.Root>
      {#if running && models.length > 1}<p
          class="mt-1 text-[11px] leading-relaxed text-muted-foreground"
        >
          Stop to change the loaded model.
        </p>{/if}
    </div>
    {#if view.backend || status?.version}<p
        class="break-words text-[11px] text-muted-foreground"
      >
        {[view.backend, status?.version].filter(Boolean).join(" · ")}
      </p>{/if}
  {/if}
  <div
    class={sidebar
      ? "flex flex-wrap items-center gap-x-3 gap-y-1"
      : "flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-hairline pt-2"}
  >
    <Button
      variant="link"
      size="xs"
      class="h-auto min-h-5 p-0 text-[11px]"
      onclick={manage}>Manage runtime</Button
    >
    {#if row}<Button
        variant="link"
        size="xs"
        class="h-auto min-h-5 p-0 text-[11px]"
        disabled={!!layout && !layout.bottomAvailable.current}
        title={layout && !layout.bottomAvailable.current
          ? "Make the window taller to show runtime output."
          : undefined}
        onclick={output}>View output</Button
      >{/if}
  </div>
</div>

{#snippet runtimeStatus()}
  <p
    class="flex min-w-0 items-start gap-1.5 text-xs leading-relaxed"
    role="status"
    aria-label="Local runtime status"
  >
    {#if operating || (!row && runtime.loading)}
      <LoaderCircleIcon
        class="mt-0.5 size-3.5 shrink-0 animate-spin text-accent-text motion-reduce:animate-none"
        aria-hidden="true"
      />
    {:else}<span
        class="mt-1.5 size-1.5 shrink-0 rounded-full {problem
          ? 'bg-destructive'
          : view.ready
            ? 'bg-success'
            : 'bg-meter-rest'}"
        aria-hidden="true"
      ></span>{/if}
    <span
      class="min-w-0 break-words font-medium {problem
        ? 'text-destructive'
        : view.ready && !operating
          ? 'text-success'
          : 'text-secondary-foreground'}">{label}</span
    >
  </p>
{/snippet}

<style>
  .runtime-sidebar {
    border-left: 1px solid var(--hairline);
    padding: 0.125rem 0 0.125rem 0.5rem;
  }
</style>
