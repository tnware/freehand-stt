<script lang="ts">
  import { session } from "$lib/stores/session.svelte";
  import * as Kbd from "$lib/components/ui/kbd";
  import { shortcutKeyLabels, shortcutSpokenLabel } from "$lib/utils/shortcuts";

  let {
    value,
    platform = session.editor.applied?.platform ?? "windows",
    label = "Keyboard shortcut",
    emptyLabel = "Not configured",
  }: {
    value: string;
    platform?: string;
    label?: string;
    emptyLabel?: string;
  } = $props();

  const keys = $derived(shortcutKeyLabels(value, platform));
  const spoken = $derived(shortcutSpokenLabel(value, platform));
</script>

{#if keys.length > 0}
  <Kbd.Group role="img" aria-label={`${label}: ${spoken}`}>
    {#each keys as key, index (`${key}-${index}`)}
      <Kbd.Root aria-hidden="true">{key}</Kbd.Root>
      {#if index < keys.length - 1}
        <span aria-hidden="true" class="text-[10px] text-muted-foreground"
          >+</span
        >
      {/if}
    {/each}
  </Kbd.Group>
{:else}
  <span class="text-xs text-muted-foreground">{emptyLabel}</span>
{/if}
