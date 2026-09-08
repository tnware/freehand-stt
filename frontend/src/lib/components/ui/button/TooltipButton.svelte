<script lang="ts">
  import * as Tooltip from "$lib/components/ui/tooltip";
  import { buttonVariants, type ButtonVariant, type ButtonSize } from "./index";
  import { Tooltip as TooltipPrimitive } from "bits-ui";
  import { cn } from "$lib/utils";

  let {
    label,
    children,
    class: className,
    variant = "ghost",
    size = "icon-xs",
    ...rest
  }: Omit<TooltipPrimitive.TriggerProps, "child" | "aria-label"> & {
    label: string;
    variant?: ButtonVariant;
    size?: ButtonSize;
  } = $props();
</script>

<Tooltip.Provider delayDuration={350}>
  <Tooltip.Root>
    <Tooltip.Trigger
      {...rest}
      aria-label={label}
      class={cn(buttonVariants({ variant, size }), className)}
    >
      {@render children?.()}
    </Tooltip.Trigger>
    <Tooltip.Content role="tooltip" sideOffset={6}>{label}</Tooltip.Content>
  </Tooltip.Root>
</Tooltip.Provider>
