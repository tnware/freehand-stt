<script lang="ts">
  import ClipboardIcon from "@lucide/svelte/icons/clipboard";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Button } from "$lib/components/ui/button";
  import Stage, { type StageTone } from "./Stage.svelte";
  import StagePick from "./StagePick.svelte";

  let {
    ordinal = "04",
    label = "Deliver",
    target,
    meta = "",
    tone = "idle",
    notice = "",
    noticeTitle = "",
    canCopy = false,
    copying = false,
    onCopy,
    onOpen,
    pickLabel = "Delivery",
  }: {
    ordinal?: string;
    label?: string;
    target: string;
    meta?: string;
    tone?: StageTone;
    /** Why delivery stopped short, in the user's terms. */
    notice?: string;
    noticeTitle?: string;
    canCopy?: boolean;
    copying?: boolean;
    onCopy: () => void;
    onOpen: () => void;
    pickLabel?: string;
  } = $props();
</script>

<Stage {ordinal} {label} {tone}>
  <StagePick value={target} {meta} label={pickLabel} {onOpen} />

  {#if notice}
    <!-- The one place the app refuses to act on its own. Saying so here, at
         the stage that would have acted, is the whole argument for the chain. -->
    <div
      class="notice rounded-md border px-2.5 py-2"
      data-tone={tone}
      role="status"
    >
      <p class="flex items-center gap-1.5 text-[11px] font-semibold">
        <TriangleAlertIcon class="size-3.5 shrink-0" aria-hidden="true" />
        {noticeTitle || "Needs you"}
      </p>
      <p class="mt-1 text-[11px] leading-snug text-secondary-foreground">
        {notice}
      </p>
    </div>
  {/if}

  {#snippet footer()}
    {#if canCopy}
      <Button size="xs" onclick={onCopy} disabled={copying}>
        <ClipboardIcon class="size-3" />
        Copy
      </Button>
    {:else}
      <span></span>
    {/if}
  {/snippet}
</Stage>

<style>
  .notice {
    border-color: color-mix(in srgb, var(--warning) 26%, transparent);
    background: color-mix(in srgb, var(--warning) 9%, transparent);
    color: var(--warning);
  }
  .notice[data-tone="bad"] {
    border-color: color-mix(in srgb, var(--destructive) 30%, transparent);
    background: color-mix(in srgb, var(--destructive) 9%, transparent);
    color: var(--destructive);
  }
</style>
