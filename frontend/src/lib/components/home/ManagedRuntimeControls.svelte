<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
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
  let now = $state(Date.now());
  $effect(() => {
    const timer = setInterval(() => {
      now = Date.now();
    }, 1000);
    return () => clearInterval(timer);
  });
  const row = $derived(runtime.statusFor(instanceID));
  const status = $derived(row?.status);
  const view = $derived(runtimePresentation(status, now));
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
  <div class="space-y-3 rounded-lg border border-hairline bg-secondary/50 p-3">
    <p
      class="flex min-w-0 flex-wrap items-center gap-2 text-xs text-muted-foreground"
      role="status"
    >
      {#if operating}<LoaderCircleIcon
          class="size-3 shrink-0 animate-spin"
        />{/if}
      <StatusBadge
        tone={problem
          ? "danger"
          : operating
            ? "accent"
            : view.ready
              ? "success"
              : "neutral"}
      >
        {runtime.pendingFor(instanceID) || view.label}
      </StatusBadge>
      {#if view.backend}<span>{view.backend}</span>{/if}
      {#if view.startup}<span class="w-full leading-relaxed"
          >{view.startup}</span
        >{/if}
    </p>
    <div class="flex flex-wrap items-center gap-2">
      <Button
        variant="outline"
        size="icon-sm"
        aria-label="Manage runtime"
        title="Manage runtime"
        onclick={onManage}><SettingsIcon class="size-4" /></Button
      >
      {#if row}<Button
          variant="outline"
          size="sm"
          onclick={() => void runtime.openOutput(instanceID)}
          >View output</Button
        >{/if}
      {#if operating}
        <Button
          variant="outline"
          size="sm"
          disabled={disabled || runtime.pendingFor(instanceID) === "Cancelling"}
          onclick={() => void runtime.cancel(instanceID)}>Cancel</Button
        >
      {:else if status?.state === "running"}
        <Button
          variant="outline"
          size="sm"
          disabled={locked}
          onclick={() => void runtime.run(instanceID, "Stop")}
          ><SquareIcon class="size-3.5" />Stop</Button
        >
      {:else}
        <Button
          variant="soft"
          size="sm"
          disabled={locked || !view.installed || !selected?.installed}
          onclick={() => void runtime.run(instanceID, "Start")}
          ><PlayIcon class="size-3.5" />Start</Button
        >
      {/if}
    </div>
  </div>

  {#if !view.installed || !selected?.installed}<Button
      variant="soft"
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
