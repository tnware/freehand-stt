<script lang="ts">
  import type { Snippet } from "svelte";
  import type { ConnectionResult } from "$lib/state";
  import { CheckStatus } from "$bindings/connection";
  import { connectionStatusLabel } from "$lib/utils/connection";
  import SettingsDisclosure from "./SettingsDisclosure.svelte";
  import ConnectionDiagnostics from "./ConnectionDiagnostics.svelte";

  let {
    connection,
    stale = false,
    busy = false,
    onCheck,
    children,
  }: {
    connection: ConnectionResult | null;
    stale?: boolean;
    busy?: boolean;
    onCheck: () => void;
    children: Snippet;
  } = $props();
  const attention = $derived(
    connection &&
      (!connection.reachable ||
        connection.checks?.some((check) => check.status === CheckStatus.CheckAttention)),
  );
  const description = $derived(
    stale
      ? "Timeout · Settings changed; check the connection again."
      : attention
        ? `Timeout · Connection needs attention (${connectionStatusLabel(connection)}).`
        : "Timeout and connection check",
  );
</script>

<SettingsDisclosure title="Request settings" {description}>
  {@render children()}
  {#if connection}
    <div class="p-5">
      <ConnectionDiagnostics result={connection} {stale} {busy} {onCheck} />
    </div>
  {/if}
</SettingsDisclosure>
