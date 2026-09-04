<script lang="ts">
  import type { Snippet } from "svelte";
  import { Label } from "$lib/components/ui/label";

  let {
    id,
    label,
    hint = "",
    control,
    action,
  }: {
    id: string;
    label: string;
    hint?: string;
    /** The editable value. Rendered borderless so it reads as a value. */
    control: Snippet;
    /** Optional status or action that directly affects this value. */
    action?: Snippet;
  } = $props();

  const uid = $props.id();
  const labelID = `${uid}-label`;
  const hintID = `${uid}-hint`;
</script>

<div
  class="grid gap-3 px-5 py-[15px] sm:grid-cols-[minmax(0,0.9fr)_minmax(240px,1.2fr)] sm:items-center sm:gap-5"
  role="group"
  aria-labelledby={labelID}
  aria-describedby={hint ? hintID : undefined}
>
  <div class="min-w-0">
    <Label id={labelID} for={id} class="text-sm font-medium">{label}</Label>
    {#if hint}
      <p id={hintID} class="mt-1 text-[12.5px] leading-relaxed text-muted-foreground">{hint}</p>
    {/if}
  </div>
  <div class="flex min-w-0 items-center justify-end gap-2">
    <div class="min-w-0 flex-1">{@render control()}</div>
    {#if action}<div class="flex shrink-0 items-center gap-2">{@render action()}</div>{/if}
  </div>
</div>
