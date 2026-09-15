<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import KeyboardIcon from "@lucide/svelte/icons/keyboard";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import type { ShortcutFeedback } from "$lib/stores/shortcutCapture.svelte";

  let {
    id,
    platform = "windows",
    title,
    description,
    requirement,
    value,
    preview = "",
    capturing = false,
    disabled = false,
    captureUnavailable = false,
    clearable = false,
    restorable = false,
    feedback = null,
    onRecord,
    onCancel,
    onClear,
    onRestore,
  }: {
    id: string;
    platform?: string;
    title: string;
    description: string;
    requirement: string;
    value: string;
    preview?: string;
    capturing?: boolean;
    disabled?: boolean;
    captureUnavailable?: boolean;
    clearable?: boolean;
    restorable?: boolean;
    feedback?: ShortcutFeedback | null;
    onRecord: () => void;
    onCancel: () => void;
    onClear?: () => void;
    onRestore: () => void;
  } = $props();

  const displayedValue = $derived(capturing ? preview : value);
  const feedbackTone = $derived(
    feedback?.state === "error"
      ? "text-destructive"
      : feedback?.state === "captured" || feedback?.state === "saved"
        ? "text-success"
        : "text-muted-foreground",
  );
</script>

<div
  class={capturing
    ? "@container/shortcut bg-accent-wash px-4 py-2.5 transition-colors"
    : "@container/shortcut px-4 py-2.5 transition-colors"}
  aria-busy={capturing}
  role="group"
  aria-labelledby={`${id}-title`}
  aria-describedby={`${id}-description`}
>
  <div
    class="flex flex-col gap-2 @min-[560px]/shortcut:flex-row @min-[560px]/shortcut:items-center @min-[560px]/shortcut:justify-between"
  >
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <p id={`${id}-title`} class="text-[13px] font-medium">{title}</p>
        <span
          class="rounded-sm bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground"
        >
          {clearable ? "Optional" : "Required"}
        </span>
      </div>
      <p
        id={`${id}-description`}
        class="mt-0.5 max-w-md text-[11.5px] leading-[1.45] text-muted-foreground"
      >
        {description}
      </p>
    </div>
    <div class="flex min-w-0 max-w-full shrink-0 flex-wrap items-center gap-2">
      <div {id} class="mr-1" aria-live="polite">
        <ShortcutKeys
          {platform}
          value={displayedValue}
          label={`${title} shortcut`}
          emptyLabel={capturing ? "Waiting for keys" : "Not configured"}
        />
      </div>
      {#if restorable && !capturing}
        <Button
          variant="ghost"
          size="icon-sm"
          onclick={onRestore}
          {disabled}
          aria-label={`Restore recommended ${title} shortcut`}
          title="Restore recommended shortcut"
        >
          <RotateCcwIcon />
        </Button>
      {/if}
      {#if clearable && value && !capturing}
        <Button
          variant="ghost"
          size="icon-sm"
          onclick={onClear}
          {disabled}
          aria-label={`Clear ${title} shortcut`}
          title="Clear shortcut"
        >
          <XIcon />
        </Button>
      {/if}
      <Button
        variant={capturing ? "default" : "soft"}
        size="sm"
        onclick={capturing ? onCancel : onRecord}
        disabled={disabled || (!capturing && captureUnavailable)}
        aria-describedby={`${id}-requirement`}
      >
        {#if capturing}
          <LoaderCircleIcon class="animate-spin" />
          Cancel capture
        {:else}
          <KeyboardIcon />
          Record
        {/if}
      </Button>
    </div>
  </div>
  <details class="group/keys mt-1.5">
    <summary
      class="inline-flex cursor-pointer list-none items-center gap-1 text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden"
    >
      Allowed keys
      <ChevronDownIcon
        class="size-3 transition-transform group-open/keys:rotate-180 motion-reduce:transition-none"
      />
    </summary>
    <p
      id={`${id}-requirement`}
      class="mt-1.5 max-w-xl text-[11.5px] leading-[1.45] text-muted-foreground"
    >
      {requirement}
    </p>
  </details>
  <div class="text-xs font-medium" aria-live="polite" aria-atomic="true">
    {#if capturing}
      <p class="mt-2 text-accent-text" role="status">
        Listening for this chord. Press Escape to cancel; captured keys will not
        trigger an action.
      </p>
    {:else if feedback}
      <p
        class={`mt-2 ${feedbackTone}`}
        role={feedback.state === "error" ? "alert" : "status"}
      >
        {feedback.message}
      </p>
    {/if}
  </div>
</div>
