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
  <Popover.Content
    role="dialog"
    aria-label={label}
    class="flex min-h-0 flex-col overflow-hidden p-0"
  >
    <p
      class="shrink-0 px-4 pt-4 pb-3 text-sm font-medium [overflow-wrap:anywhere]"
    >
      {title}
    </p>
    <!-- Focus the message before the recovery action so long errors open at the start. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      tabindex="0"
      role="region"
      aria-label="Details message"
      class="min-h-0 overflow-y-auto overscroll-contain px-4 pb-4 text-[13px] leading-relaxed whitespace-pre-wrap [overflow-wrap:anywhere] focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
    >
      {message}
    </div>
    {#if actionLabel && onAction}
      <div class="shrink-0 border-t border-hairline px-4 py-3">
        <Button
          variant="outline"
          size="sm"
          onclick={() => {
            open = false;
            onAction?.();
          }}>{actionLabel}</Button
        >
      </div>
    {/if}
  </Popover.Content>
</Popover.Root>
