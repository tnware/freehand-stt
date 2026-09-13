<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import SettingsIcon from "@lucide/svelte/icons/settings-2";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let {
    runtime,
    disabled = false,
    onManage,
  }: {
    runtime: ManagedRuntimeState;
    disabled?: boolean;
    onManage: () => void;
  } = $props();
  const uid = $props.id();
  const status = $derived(runtime.status);
  const view = $derived(runtimePresentation(status));
  const locked = $derived(
    disabled || runtime.busy || runtime.loading || !status?.supported,
  );
  const running = $derived(status?.state === "running");
  const problem = $derived(runtime.error || status?.error || "");
  const backend = $derived(
    status?.backend === "cpu"
      ? "CPU"
      : status?.backend?.startsWith("cuda")
        ? "CUDA"
        : "",
  );
  const choices = $derived(
    (status?.models ?? []).map((model) => ({
      value: model.id,
      label: model.name,
    })),
  );
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-3">
    <div class="min-w-0">
      <p class="text-xs font-medium text-muted-foreground">Connection</p>
      <p class="mt-1 text-sm">Local speech</p>
    </div>
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label="Manage local runtime"
      title="Manage local runtime"
      onclick={onManage}
    >
      <SettingsIcon class="size-4" />
    </Button>
  </div>
  <div class="space-y-1.5">
    <label for={`${uid}-model`} class="text-xs font-medium">Model</label>
    <Select.Root
      type="single"
      value={status?.selectedModel ?? ""}
      items={choices}
      disabled={locked}
      onValueChange={(id) => {
        if (!locked) void runtime.useModel(id);
      }}
    >
      <Select.Trigger id={`${uid}-model`} class="w-full">
        <span class="truncate"
          >{view.selected?.name ?? "Choose a downloaded model"}</span
        >
      </Select.Trigger>
      <Select.Content>
        {#each status?.models ?? [] as model (model.id)}
          <Select.Item
            value={model.id}
            label={model.name}
            disabled={!model.installed}
          >
            {model.name}{!model.installed ? " · Not downloaded" : ""}
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>
  <div class="flex items-center justify-between gap-3">
    <p
      class="flex min-w-0 items-center gap-2 text-xs text-muted-foreground"
      role="status"
      aria-live="polite"
    >
      {#if runtime.busy}<LoaderCircleIcon
          class="size-3 shrink-0 animate-spin"
        />{/if}
      <span
        >{runtime.busy
          ? runtime.pending || view.label
          : view.ready
            ? "Model ready"
            : view.label}{backend ? ` · ${backend}` : ""}</span
      >
    </p>
    <div class="flex shrink-0 items-center gap-1">
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label="Refresh runtime status"
        title="Refresh runtime status"
        disabled={locked}
        onclick={() => void runtime.run("RefreshCatalog")}
        ><RefreshCwIcon class="size-3.5" /></Button
      >
      {#if runtime.busy}
        <Button
          variant="ghost"
          size="sm"
          disabled={disabled || runtime.pending === "Cancelling"}
          onclick={() => void runtime.cancel()}>Cancel</Button
        >
      {:else if running}
        <Button
          variant="ghost"
          size="sm"
          disabled={locked}
          onclick={() => void runtime.run("Stop")}
          ><SquareIcon class="size-3.5" />Stop runtime</Button
        >
      {:else}
        <Button
          variant="ghost"
          size="sm"
          disabled={locked ||
            !status?.enabled ||
            !view.installed ||
            !view.selected?.installed}
          onclick={() => void runtime.run("Start")}
          ><PlayIcon class="size-3.5" />Start runtime</Button
        >
      {/if}
    </div>
  </div>
  {#if !view.installed || !view.selected?.installed}
    <Button variant="outline" size="sm" class="w-full" onclick={onManage}
      >Set up local speech</Button
    >
  {/if}
  {#if problem}<p class="text-xs text-destructive" role="alert">
      {problem}
    </p>{/if}
  {#if !running && !runtime.busy && view.installed && view.selected?.installed}
    <p class="text-xs text-muted-foreground">
      Start the runtime to transcribe. No automatic server fallback.
    </p>
  {/if}
</div>
