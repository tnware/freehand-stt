<script lang="ts">
  import * as Popover from "$lib/components/ui/popover";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import { cn } from "$lib/utils";
  let {
    title,
    message,
    label = `${title} details`,
    actionLabel,
    onAction,
  }: {
    title: string;
    message: string;
    label?: string;
    actionLabel?: string;
    onAction?: () => void;
  } = $props();
  let open = $state(false);
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    aria-label={label}
    class={cn(
      buttonVariants({ variant: "ghost", size: "xs" }),
      "shrink-0 px-2 text-xs aria-expanded:bg-subtle-fill-hover",
    )}>Details</Popover.Trigger
  >
  <Popover.Content role="dialog" aria-label={label}>
    <p class="text-sm font-medium">{title}</p>
    <p class="mt-2 text-[13px] leading-relaxed break-words whitespace-pre-wrap">{message}</p>
    {#if actionLabel && onAction}
      <Button
        variant="outline"
        size="sm"
        class="mt-3"
        onclick={() => {
          open = false;
          onAction?.();
        }}>{actionLabel}</Button
      >
    {/if}
  </Popover.Content>
</Popover.Root>
