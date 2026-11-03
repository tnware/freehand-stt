<script lang="ts">
  import Disclosure from "$lib/components/common/Disclosure.svelte";
  import type { Connection } from "$bindings/savedconnection";
  import type {
    Instance,
    InstanceStatus,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import { connectionWorkflows } from "$lib/utils/connectionChoices";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { runtimeConnectionModels } from "$lib/utils/runtimeConnectionModels";
  import type { Purpose } from "$bindings/savedconnection";
  import RuntimeStatus from "$lib/components/common/RuntimeStatus.svelte";
  import { Button } from "$lib/components/ui/button";
  import WorkflowIcon from "@lucide/svelte/icons/workflow";
  import ShieldCheckIcon from "@lucide/svelte/icons/shield-check";

  let {
    connection,
    instance,
    status,
    providers,
    pending = "",
    problem = "",
    busy = false,
    onManageRuntime,
    onWorkflow,
  }: {
    connection: Connection;
    instance?: Instance;
    status?: InstanceStatus;
    providers: ProviderDescriptor[];
    pending?: string;
    problem?: string;
    busy?: boolean;
    onManageRuntime: () => void;
    onWorkflow: (purpose: Purpose) => void;
  } = $props();
  const provider = $derived(providers.find((p) => p.id === instance?.provider));
  const view = $derived(
    runtimePresentation(status?.status, undefined, pending),
  );
  const models = $derived(
    runtimeConnectionModels(instance, status?.status, provider?.models ?? []),
  );
  const uses = $derived(
    connectionWorkflows.filter((role) => connection.uses?.includes(role.id)),
  );
</script>

<div class="min-w-0 space-y-3 py-3">
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-0 flex-1">
      <p class="content-kicker">Built-in connection</p>
      <p class="content-meta mt-1">
        {provider?.name ?? instance?.provider ?? "Unavailable runtime"} · Runtime-owned
      </p>
    </div>
    <RuntimeStatus {view} {problem} badge />
  </div>
  <dl class="border-y border-hairline py-3">
    <dt class="content-kicker">Selected model</dt>
    <dd class="content-value mt-1 break-words">
      {models.primary?.name ?? "No model selected"}
    </dd>
    {#if models.primary}<dd class="content-meta mt-1">
        {models.primary.label}
      </dd>{/if}
    {#if models.speech}
      <dt class="content-kicker mt-3">Selected speech model</dt>
      <dd class="content-value mt-1 break-words">{models.speech.name}</dd>
      <dd class="content-meta mt-1">{models.speech.label}</dd>
    {/if}
  </dl>
  <p class="content-meta">
    {uses.map((role) => role.label).join(" · ") ||
      "No qualified tasks for this model"}
  </p>
  {#if models.primary?.missing || models.speech?.missing}
    <p class="content-meta">
      Download missing models in Local runtime before starting.
    </p>
  {/if}
  <div class="flex flex-wrap items-center justify-between gap-3">
    <p class="content-meta">Local endpoint · No connection credentials</p>
    <Button
      variant="outline"
      size="xs"
      disabled={busy}
      onclick={onManageRuntime}>Manage runtime</Button
    >
  </div>
</div>
<Disclosure
  title="Connection ownership & safety"
  icon={ShieldCheckIcon}
  class="border-t border-hairline"
>
  <div class="content-meta space-y-3 pb-3">
    <p>
      Available automatically from this local runtime. Its name, transport,
      model, and supported tasks are managed by the runtime, not by a saved
      server configuration.
    </p>
    <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
      <dt>Name</dt>
      <dd class="content-value break-words">{connection.name}</dd>
      <dt>Installed binary</dt>
      <dd class="content-value">{view.backend || "Not reported"}</dd>
      {#if models.primary}<dt>Model ID</dt>
        <dd class="break-all font-mono text-foreground">
          {models.primary.id}
        </dd>{/if}
      {#if models.speech}<dt>Speech model ID</dt>
        <dd class="break-all font-mono text-foreground">
          {models.speech.id}
        </dd>{/if}
    </dl>
    <p>
      Stopping the runtime keeps task selections. Requests never fall back to a
      remote server.
    </p>
  </div>
</Disclosure>
<section class="space-y-3 border-t border-hairline py-3">
  <h3 class="content-section-title flex items-center gap-2">
    <WorkflowIcon class="content-section-icon" aria-hidden="true" />Task
    settings
  </h3>
  <p class="content-meta">
    Choose this connection in each task. Language, recording mode, cleanup
    intent, and other task options stay with that task.
  </p>
  <div class="flex flex-wrap gap-2">
    {#each uses as role (role.id)}
      <Button
        variant="outline"
        size="xs"
        disabled={busy}
        onclick={() => onWorkflow(role.id)}>{role.label} settings</Button
      >
    {/each}
  </div>
</section>
