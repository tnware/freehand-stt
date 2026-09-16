<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let {
    label,
    disabled = false,
    onclick,
  }: {
    label: string;
    disabled?: boolean;
    onclick: () => void;
  } = $props();
</script>

<button
  type="button"
  class="flex min-h-8 w-full items-center justify-between gap-2 border-t border-hairline py-1 text-left text-xs text-secondary-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
  {disabled}
  {onclick}
>
  <span class="min-w-0">{label}</span>
  <ChevronRightIcon class="size-3 shrink-0" aria-hidden="true" />
</button>
