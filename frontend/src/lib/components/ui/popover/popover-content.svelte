<script lang="ts">
  import { Popover } from "bits-ui";
  import { cn } from "$lib/utils";
  let {
    class: className,
    sideOffset = 8,
    align = "start",
    ...props
  }: Popover.ContentProps = $props();
</script>

<Popover.Portal>
  <Popover.Content
    {sideOffset}
    {align}
    collisionPadding={12}
    class={cn(
      "z-50 w-96 max-w-[calc(100vw-24px)] max-h-(--bits-popover-content-available-height) overflow-y-auto rounded-lg border border-border bg-popover p-4 text-popover-foreground shadow-xl outline-none",
      className,
    )}
    {...props}
  />
</Popover.Portal>
