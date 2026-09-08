<script lang="ts">
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  import * as Popover from "$lib/components/ui/popover";
  import { buttonVariants } from "$lib/components/ui/button";
  let { label, text }: { label: string; text: string } = $props();
</script>

<Popover.Root>
  <Popover.Trigger aria-label={label} class={buttonVariants({ variant: "ghost", size: "icon-xs" })}>
    <CircleHelpIcon class="size-3.5 text-muted-foreground" />
  </Popover.Trigger>
  <Popover.Content role="dialog" aria-label={label} class="w-80">
    <p class="text-sm leading-relaxed whitespace-pre-wrap">{text}</p>
  </Popover.Content>
</Popover.Root>
