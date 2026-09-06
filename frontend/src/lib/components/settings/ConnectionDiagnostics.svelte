<script lang="ts">
  import { CheckKind, CheckStatus } from "$bindings/connection";
  import type { ConnectionResult } from "$lib/state";
  import { connectionDescription } from "$lib/utils/connection";
  import { Button } from "$lib/components/ui/button";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  let {
    result,
    stale = false,
    busy = false,
    onCheck,
  }: { result: ConnectionResult; stale?: boolean; busy?: boolean; onCheck?: () => void } = $props();
  const labels: Record<CheckKind, string> = {
    [CheckKind.$zero]: "Check",
    connection: "Connection",
    authentication: "Authentication",
    model: "Selected model",
    configuration: "Configuration",
  };
</script>

<section class="space-y-3" aria-label="Connection check results" aria-live="polite">
  <div class="flex items-center justify-between gap-3">
    <div>
      <p class="text-sm font-medium">Connection check</p>
      <p class="mt-1 text-xs text-muted-foreground">
        {stale
          ? "Settings changed. Check again for current results."
          : "Metadata only. No model was invoked."}
      </p>
    </div>
    {#if onCheck}<Button variant="ghost" size="sm" onclick={onCheck} disabled={busy}
        >{busy ? "Checking…" : "Check again"}</Button
      >{/if}
  </div>
  {#if stale}<p class="text-xs text-muted-foreground">
      Previous results apply to the settings that were tested.
    </p>
  {:else if result.checks?.length}<dl class="space-y-3">
      {#each result.checks as check (check.kind)}<div class="grid grid-cols-[18px_1fr] gap-x-2">
          {#if check.status === CheckStatus.CheckPassed}<CircleCheckIcon
              class="mt-0.5 size-4 text-success"
            />{:else if check.status === CheckStatus.CheckAttention}<CircleAlertIcon
              class="mt-0.5 size-4 text-warning"
            />{:else}<CircleHelpIcon class="mt-0.5 size-4 text-muted-foreground" />{/if}
          <div>
            <dt class="text-xs text-muted-foreground">{labels[check.kind]}</dt>
            <dd class="mt-0.5 text-sm">{check.summary}</dd>
            {#if check.detail && check.status !== CheckStatus.CheckPassed}<dd
                class="mt-1 text-xs leading-relaxed text-muted-foreground"
              >
                {check.detail}
              </dd>{/if}
          </div>
        </div>{/each}
    </dl>{:else}<p class="text-xs text-muted-foreground">{connectionDescription(result)}</p>{/if}
  {#if !stale}
    <div
      class="space-y-1 border-t border-hairline pt-3 text-[11px] leading-relaxed text-muted-foreground"
    >
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
