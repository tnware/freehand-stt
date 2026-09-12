<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    compact = false,
    description = "",
    controlID,
    control,
    children,
  }: {
    title: string;
    compact?: boolean;
    description?: string;
    controlID?: string;
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
  class={compact ? "px-5 py-3" : "px-5 py-4"}
  role="group"
  aria-labelledby={titleID}
  aria-describedby={description ? descriptionID : undefined}
>
  <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
    <div class="min-w-0 flex-[1_1_12rem]">
      {#if controlID}
        <label id={titleID} for={controlID} class="cursor-pointer text-sm font-medium"
          >{title}</label
        >
      {:else}
        <p id={titleID} class="text-sm font-medium">{title}</p>
      {/if}
      {#if description}
        <p id={descriptionID} class="mt-1 text-[13px] leading-relaxed text-muted-foreground">
          {description}
        </p>
      {/if}
    </div>
    {#if control}
      <div class="ml-auto flex max-w-full shrink-0 flex-wrap items-center gap-2">
        {@render control()}
      </div>
    {/if}
  </div>
  {#if children}
    <div class="mt-3.5">{@render children()}</div>
  {/if}
</div>
