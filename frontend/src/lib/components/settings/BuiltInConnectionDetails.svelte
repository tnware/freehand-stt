<script lang="ts">
  import type { Connection } from "$bindings/savedconnection";
  import type {
    Instance,
    InstanceStatus,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import { connectionWorkflows } from "$lib/utils/connectionChoices";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import type { Purpose } from "$bindings/savedconnection";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
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
  const view = $derived(runtimePresentation(status?.status));
  const model = $derived(
    provider?.models?.find((m) => m.id === instance?.model),
  );
  const uses = $derived(
    connectionWorkflows.filter((role) => connection.uses?.includes(role.id)),
  );
</script>

<SettingsCard>
  <div class="space-y-3 py-3">
    <div class="flex flex-wrap items-center gap-3">
      <span class="flex size-8 shrink-0 items-center justify-center">
        <ProviderIcon
          profile={instance?.provider ??
            connection.details.compatibilityProfile}
          size={20}
        />
      </span>
      <div class="min-w-0 flex-1">
        <h3 class="text-[13px] font-medium">
          {provider?.name ?? instance?.provider ?? "Unavailable runtime"}
        </h3>
        <p class="mt-1 text-xs text-secondary-foreground">
          Built-in connection · Runtime-owned
        </p>
      </div>
      <StatusBadge
        tone={status?.status.state === "running"
          ? "success"
          : status?.status.state === "error"
            ? "danger"
            : "neutral"}
        dot>{view.label}</StatusBadge
      >
    </div>
    <div class="border-y border-hairline py-3">
      <p class="text-xs font-medium text-secondary-foreground">
        Selected model
      </p>
      <p class="mt-1 break-words text-[13px] font-medium">
        {model?.name ?? instance?.model ?? "Unavailable"}
      </p>
      <p class="mt-1 text-xs text-secondary-foreground">
        {uses.map((role) => role.label).join(" · ") ||
          "No qualified tasks for this model"}
      </p>
    </div>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-xs text-secondary-foreground">
        Local endpoint · No connection credentials
      </p>
      <Button variant="soft" disabled={busy} onclick={onManageRuntime}
        >Manage runtime</Button
      >
    </div>
    <details class="border-t border-hairline pt-3">
      <summary
        class="w-fit cursor-pointer rounded-sm text-xs font-medium text-accent-text focus-visible:outline-ring"
        >Connection ownership &amp; safety</summary
      >
      <div
        class="mt-3 space-y-3 text-xs leading-relaxed text-secondary-foreground"
      >
        <p>
          Available automatically from this local runtime. Its name, transport,
          model, and supported tasks are managed by the runtime, not by a saved
          server configuration.
        </p>
        <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
          <dt>Name</dt>
          <dd class="break-words text-foreground">{connection.name}</dd>
          <dt>Installed binary</dt>
          <dd class="text-foreground">{view.backend || "Not reported"}</dd>
          {#if model}<dt>Model ID</dt>
            <dd class="break-all font-mono text-foreground">{model.id}</dd>{/if}
        </dl>
        <p>
          Stopping the runtime keeps task selections. Requests never fall back
          to a remote server.
        </p>
      </div>
    </details>
  </div>
</SettingsCard>
<SettingsCard>
  <div class="space-y-3 py-3">
    <h3 class="text-[13px] font-medium">Task settings</h3>
    <p class="text-xs leading-relaxed text-secondary-foreground">
      Choose this connection in each task. Language, recording mode, cleanup
      intent, and other task options stay with that task.
    </p>
    <div class="flex flex-wrap gap-2">
      {#each uses as role (role.id)}
        <Button
          variant="outline"
          size="sm"
          disabled={busy}
          onclick={() => onWorkflow(role.id)}>{role.label} settings</Button
        >
      {/each}
    </div>
  </div>
</SettingsCard>
