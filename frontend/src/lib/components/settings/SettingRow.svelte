<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    description = "",
    control,
    children,
  }: {
    title: string;
    description?: string;
    /** Trailing control: a switch, a button, a segmented group. */
    control?: Snippet;
    /** Full-width content below the title, for dependent fields. */
    children?: Snippet;
  } = $props();

  const uid = $props.id();
  const titleID = `${uid}-title`;
  const descriptionID = `${uid}-description`;
</script>

<div
  class="px-5 py-[15px]"
  role="group"
  aria-labelledby={titleID}
  aria-describedby={description ? descriptionID : undefined}
>
  <div class="flex items-center justify-between gap-5">
    <div class="min-w-0">
      <p id={titleID} class="text-sm font-medium">{title}</p>
      {#if description}
        <p id={descriptionID} class="mt-1 text-[12.5px] leading-relaxed text-muted-foreground">{description}</p>
      {/if}
    </div>
    {#if control}
      <div class="shrink-0">{@render control()}</div>
    {/if}
  </div>
  {#if children}
    <div class="mt-3.5">{@render children()}</div>
  {/if}
</div>
