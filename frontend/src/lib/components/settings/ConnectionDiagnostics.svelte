<script lang="ts">
  import { session } from "$lib/stores/session.svelte";
  import { CheckKind, CheckStatus } from "$bindings/connection";
  import type { ConnectionResult } from "$lib/state";
  import { connectionDescription } from "$lib/utils/connection";
  import { Button } from "$lib/components/ui/button";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  let {
    result,
    platform = session.editor.applied?.platform ?? "windows",
    stale = false,
    busy = false,
    onCheck,
  }: {
    result: ConnectionResult;
    platform?: string;
    stale?: boolean;
    busy?: boolean;
    onCheck?: () => void;
  } = $props();
  const labels: Record<CheckKind, string> = {
    [CheckKind.$zero]: "Check",
    connection: "Connection",
    authentication: "Authentication",
    model: "Selected model",
    configuration: "Configuration",
  };
</script>

<section
  class="space-y-3"
  aria-label="Connection check results"
  aria-live="polite"
>
  <div class="flex items-center justify-between gap-3">
    <div>
      <p class="content-section-title">Connection check</p>
      <p class="content-meta mt-1">
        {stale
          ? "Settings changed. Check again for current results."
          : "Metadata only. No model was invoked."}
      </p>
    </div>
    {#if onCheck}<Button
        variant="soft"
        size="sm"
        onclick={onCheck}
        disabled={busy}>{busy ? "Checking…" : "Check again"}</Button
      >{/if}
  </div>
  {#if stale}<p class="content-meta">
      Previous results apply to the settings that were tested.
    </p>
  {:else if result.checks?.length}<dl
      class="divide-y divide-hairline border-y border-hairline"
    >
      {#each result.checks as check (check.kind)}<div
          class="grid grid-cols-[18px_1fr] gap-x-2 py-2.5"
        >
          {#if check.status === CheckStatus.CheckPassed}<CircleCheckIcon
              class="mt-0.5 size-4 text-success"
            />{:else if check.status === CheckStatus.CheckAttention}<CircleAlertIcon
              class="mt-0.5 size-4 text-warning"
            />{:else}<CircleHelpIcon
              class="mt-0.5 size-4 text-muted-foreground"
            />{/if}
          <div>
            <dt class="content-kicker">
              {labels[check.kind]}
            </dt>
            <dd class="content-value mt-1">{check.summary}</dd>
            {#if check.detail && check.status !== CheckStatus.CheckPassed}<dd
                class="content-meta mt-1"
              >
                {check.detail}
              </dd>{/if}
          </div>
        </div>{/each}
    </dl>{:else}<p class="content-meta">
      {connectionDescription(result, platform)}
    </p>{/if}
  {#if !stale}
    {#if result.serverVersion}<p class="content-meta">
        Server version: {result.serverVersion}
      </p>{/if}
    {#if result.models?.some((model) => model.capability)}
      <div class="space-y-1">
        <p class="content-kicker">Advertised server models</p>
        {#each result.models.filter((model) => model.capability) as model (model.id)}
          <p class="content-meta break-words">
            {model.id} · {model.capability}{model.device
              ? ` · ${model.device}`
              : ""}
          </p>
        {/each}
        <p class="content-meta">
          Model choices use the task’s advertised capability. Choose the model
          behavior profile separately.
        </p>
      </div>
    {/if}
    <div class="content-meta space-y-1 border-t border-hairline pt-3">
      <p>Inference support and inference authorization remain unverified.</p>
      {#if result.httpStatus || result.latencyMilliseconds > 0}
        <p class="flex flex-wrap gap-x-3">
          {#if result.httpStatus}<span>HTTP {result.httpStatus}</span>{/if}
          {#if result.latencyMilliseconds > 0}<span
              >{result.latencyMilliseconds.toLocaleString()} ms</span
            >{/if}
        </p>
      {/if}
    </div>
  {/if}
</section>
