<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import SettingsIcon from "@lucide/svelte/icons/settings-2";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let {
    runtime,
    instanceID,
    disabled = false,
    onManage,
  }: {
    runtime: ManagedRuntimeState;
    instanceID: string;
    disabled?: boolean;
    onManage: () => void;
  } = $props();
  const uid = $props.id();
  const row = $derived(runtime.statusFor(instanceID));
  const status = $derived(row?.status);
  const view = $derived(runtimePresentation(status));
  const provider = $derived(
    runtime.providers.find((p) => p.id === row?.instance.provider),
  );
  const operating = $derived(runtime.isBusy(instanceID));
  const locked = $derived(
    disabled || operating || runtime.loading || !status?.supported,
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
    runtime.errorFor(instanceID) || runtime.error || status?.error || "",
  );
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-3">
    <div class="min-w-0">
      <p class="truncate text-sm font-medium">
        {row?.instance.name ?? "Runtime unavailable"}
      </p>
      <p class="text-xs text-muted-foreground">
        {provider?.name ?? row?.instance.provider ?? "Status unavailable"}
      </p>
    </div>
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label="Manage runtime"
      title="Manage runtime"
      onclick={onManage}><SettingsIcon class="size-4" /></Button
    >
  </div>
  {#if row}
    <div class="space-y-1.5">
      <label for={`${uid}-model`} class="text-xs font-medium"
        >Selected model</label
      >
      <Select.Root
        type="single"
        value={row.instance.model}
        items={choices}
        disabled={locked || !models.length}
        onValueChange={(model) => {
          if (!locked && models.some((m) => m.id === model))
            void runtime.saveInstance({ ...row.instance, model });
        }}
      >
        <Select.Trigger id={`${uid}-model`} class="w-full"
          ><span class="truncate">{selected?.name ?? row.instance.model}</span
          ></Select.Trigger
        >
        <Select.Content
          >{#each models as model (model.id)}<Select.Item
              value={model.id}
              label={model.name}>{model.name}</Select.Item
            >{/each}</Select.Content
        >
      </Select.Root>
    </div>
  {/if}
  <div class="flex items-center justify-between gap-3">
    <p
      class="flex min-w-0 items-center gap-2 text-xs text-muted-foreground"
      role="status"
    >
      {#if operating}<LoaderCircleIcon
          class="size-3 shrink-0 animate-spin"
        />{/if}
      {runtime.pendingFor(instanceID) || view.label}
    </p>
    <div class="flex shrink-0 items-center gap-1">
      {#if operating}
        <Button
          variant="ghost"
          size="sm"
          disabled={disabled || runtime.pendingFor(instanceID) === "Cancelling"}
          onclick={() => void runtime.cancel(instanceID)}>Cancel</Button
        >
      {:else if status?.state === "running"}
        <Button
          variant="ghost"
          size="sm"
          disabled={locked}
          onclick={() => void runtime.run(instanceID, "Stop")}
          ><SquareIcon class="size-3.5" />Stop</Button
        >
      {:else}
        <Button
          variant="ghost"
          size="sm"
          disabled={locked || !view.installed || !selected?.installed}
          onclick={() => void runtime.run(instanceID, "Start")}
          ><PlayIcon class="size-3.5" />Start</Button
        >
      {/if}
    </div>
  </div>
  {#if row?.activeModel}<p class="break-all text-xs text-muted-foreground">
      Active model: {row.activeModel}
    </p>{/if}
  {#if row?.activeModel && row.activeModel !== row.instance.model}<p
      class="text-xs text-muted-foreground"
    >
      The active API model identity differs from the selected catalog key.
    </p>{/if}
  {#if !view.installed || !selected?.installed}<Button
      variant="outline"
      size="sm"
      class="w-full"
      onclick={onManage}>Set up runtime</Button
    >{/if}
  {#if problem}<p class="text-xs text-destructive" role="alert">
      {problem}
    </p>{/if}
  {#if row && !view.ready}<p class="text-xs text-muted-foreground">
      This connection stays selected while unavailable. No automatic server
      fallback.
    </p>{/if}
</div>
