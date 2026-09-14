<script lang="ts">
  import type { Connection } from "$bindings/savedconnection";
  import type {
    Instance,
    InstanceStatus,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import { connectionWorkflows } from "$lib/utils/connectionChoices";
  import type { Purpose } from "$bindings/savedconnection";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { Button } from "$lib/components/ui/button";

  let {
    connection,
    instance,
    status,
    providers,
    busy = false,
    onManageRuntime,
    onWorkflow,
  }: {
    connection: Connection;
    instance?: Instance;
    status?: InstanceStatus;
    providers: ProviderDescriptor[];
    busy?: boolean;
    onManageRuntime: () => void;
    onWorkflow: (purpose: Purpose) => void;
  } = $props();
  const provider = $derived(providers.find((p) => p.id === instance?.provider));
  const model = $derived(
    provider?.models?.find((m) => m.id === instance?.model),
  );
  const uses = $derived(
    connectionWorkflows.filter((role) => connection.uses?.includes(role.id)),
  );
</script>

<SettingsCard>
  <div class="space-y-4 px-5 py-4">
    <div>
      <p class="text-sm font-medium">Built-in connection</p>
      <p class="mt-1 text-xs text-muted-foreground">
        Available automatically from this local runtime. Its name, transport,
        model, and supported tasks are managed by the runtime, not by a saved
        server configuration.
      </p>
    </div>
    <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-5 gap-y-3 text-sm">
      <dt class="text-muted-foreground">Name</dt>
      <dd class="break-words">{connection.name}</dd>
      <dt class="text-muted-foreground">Runtime</dt>
      <dd>{provider?.name ?? instance?.provider ?? "Unavailable runtime"}</dd>
      <dt class="text-muted-foreground">Transport</dt>
      <dd>Local, runtime-owned endpoint · No connection credentials</dd>
      <dt class="text-muted-foreground">Model</dt>
      <dd class="break-words">
        {model?.name ?? instance?.model ?? "Unavailable"}{#if model}<span
            class="mt-1 block font-mono text-xs text-muted-foreground"
            >{model.id}</span
          >{/if}
      </dd>
      <dt class="text-muted-foreground">Status</dt>
      <dd>{status?.status.state ?? "Unavailable"}</dd>
      <dt class="text-muted-foreground">Supported tasks</dt>
      <dd>
        {uses.map((role) => role.label).join(" · ") ||
          "No qualified tasks for this model"}
      </dd>
    </dl>
    <p class="text-xs text-muted-foreground">
      Stopping the runtime keeps task selections. Requests never fall back to a
      remote server.
    </p>
    <Button variant="outline" disabled={busy} onclick={onManageRuntime}
      >Manage runtime</Button
    >
  </div>
</SettingsCard>
<SettingsCard>
  <div class="space-y-3 px-5 py-4">
    <h3 class="text-sm font-medium">Task settings</h3>
    <p class="text-xs text-muted-foreground">
      Choose this connection in each task. Language, recording mode, cleanup
      intent, and other task options stay with that task.
    </p>
    <div class="flex flex-wrap gap-2">
      {#each uses as role (role.id)}
        <Button
          variant="ghost"
          size="sm"
          disabled={busy}
          onclick={() => onWorkflow(role.id)}>{role.label} settings</Button
        >
      {/each}
    </div>
  </div>
</SettingsCard>
