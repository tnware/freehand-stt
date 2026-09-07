<script lang="ts">
  import { PaneGroup, type PaneGroupProps } from "paneforge";
  import { cn } from "$lib/utils";
  let { class: className, ...props }: PaneGroupProps = $props();
</script>

<PaneGroup
  class={cn("flex h-full w-full min-h-0 min-w-0", className)}
  {...props}
/>
