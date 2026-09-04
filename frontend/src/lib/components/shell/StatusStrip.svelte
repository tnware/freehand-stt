<script lang="ts">
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  import { Button } from "$lib/components/ui/button";
  import type { ConnectionResult, Settings } from "$lib/state";
  import { connectionStatusLabel, connectionSucceeded } from "$lib/utils/connection";
  import { endpointHost } from "$lib/utils/endpoint";

  let {
    settings,
    connection,
    version = "",
    aboutOpen = false,
    onAbout,
  }: {
    settings: Settings | null;
    connection: ConnectionResult | null;
    version?: string;
    aboutOpen?: boolean;
    onAbout: () => void;
  } = $props();

  const host = $derived(settings ? endpointHost(settings.baseURL) : "");
  let now = $state(Date.now());
  $effect(() => {
    const timer = setInterval(() => (now = Date.now()), 30_000);
    return () => clearInterval(timer);
  });

  const probeAge = (checkedAt: string): string => {
    const elapsed = Math.max(0, now - Date.parse(checkedAt));
    const minutes = Math.floor(elapsed / 60_000);
    if (minutes < 1) return "checked just now";
    if (minutes < 60) return `checked ${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `checked ${hours}h ago`;
    return `checked ${Math.floor(hours / 24)}d ago`;
  };

  const connectionState = $derived.by(() => {
    if (!settings) return { label: "Loading", detail: "reading settings", dot: "bg-muted-foreground" };
    if (!connection) return { label: "Configured", detail: "not checked yet", dot: "bg-muted-foreground" };
    if (connectionSucceeded(connection)) {
      return {
        label: "Reachable",
        detail: [host, probeAge(connection.checkedAt)].filter(Boolean).join(" · "),
        dot: "bg-success",
      };
    }
    return {
      label: "Unavailable",
      detail: [host, probeAge(connection.checkedAt)].filter(Boolean).join(" · "),
      dot: "bg-destructive",
    };
  });

  const connectionTitle = $derived(
    connection
      ? `${connectionStatusLabel(connection)}${host ? ` · ${host}` : ""}`
      : host || connectionState.detail,
  );
</script>

<!-- Endpoint reachability is different from dictation state. Keep it quiet,
     persistent, and truthful about whether a probe has actually run. -->
<footer
  class="flex h-8 shrink-0 items-center justify-between gap-3 border-t border-hairline bg-layer-fill px-4 text-[11px]"
>
  <div class="flex min-w-0 items-center gap-2.5" title={connectionTitle}>
    <span class="size-1.5 shrink-0 rounded-full {connectionState.dot}"></span>
    <span class="shrink-0 text-secondary-foreground">{connectionState.label}</span>
    {#if connectionState.detail}
      <span class="truncate text-ink-quiet">{connectionState.detail}</span>
    {/if}
  </div>
  <div class="flex shrink-0 items-center gap-3">
    {#if version}
      <span class="text-ink-quiet">{version}</span>
    {/if}
    <Button
      variant="ghost"
      size="xs"
      class="-mr-1.5 h-6 text-[11px] text-secondary-foreground hover:text-foreground"
      onclick={onAbout}
      aria-label={aboutOpen ? "Focus About" : "Open About"}
      title={aboutOpen ? "Focus About" : "Open About"}
    >
      <CircleHelpIcon class="size-3.5" />
      About
    </Button>
  </div>
</footer>
