<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import ButtonIcon from "$lib/components/ui/button/ButtonIcon.svelte";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

  let {
    label,
    sidebar = false,
    busy = false,
    disabled = false,
    onclick,
  }: {
    label: string;
    sidebar?: boolean;
    busy?: boolean;
    disabled?: boolean;
    onclick: () => void;
  } = $props();
</script>

<Button
  variant={sidebar ? "ghost" : "outline"}
  size={sidebar ? "icon-xs" : "sm"}
  aria-label={label}
  title={label}
  aria-busy={busy}
  disabled={disabled || busy}
  {onclick}
>
  <ButtonIcon icon={RefreshCwIcon} {busy} />
  {#if !sidebar}{label}{/if}
</Button>
