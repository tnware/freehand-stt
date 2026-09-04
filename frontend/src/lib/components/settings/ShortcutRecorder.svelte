<script lang="ts">
  import KeyboardIcon from "@lucide/svelte/icons/keyboard";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import type { ShortcutFeedback } from "$lib/stores/shortcutCapture.svelte";

  let {
    id,
    title,
    description,
    requirement,
    value,
    preview = "",
    capturing = false,
    disabled = false,
    clearable = false,
    restorable = false,
    feedback = null,
    onRecord,
    onCancel,
    onClear,
    onRestore,
  }: {
    id: string;
    title: string;
    description: string;
    requirement: string;
    value: string;
    preview?: string;
    capturing?: boolean;
    disabled?: boolean;
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
    ? "bg-primary/5 px-5 py-[19px] transition-colors"
    : "px-5 py-[19px] transition-colors"}
  aria-busy={capturing}
  role="group"
  aria-labelledby={`${id}-title`}
  aria-describedby={`${id}-description`}
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <p id={`${id}-title`} class="text-sm font-medium">{title}</p>
        <span
          class="rounded-sm bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground"
        >
          {clearable ? "Optional" : "Required"}
        </span>
      </div>
      <p
        id={`${id}-description`}
        class="mt-1 text-[13px] leading-relaxed text-muted-foreground"
      >
        {description}
      </p>
      <p class="mt-1.5 text-xs leading-relaxed text-muted-foreground/90">{requirement}</p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      <div id={id} class="mr-1" aria-live="polite">
        <ShortcutKeys
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
          disabled={disabled}
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
          disabled={disabled}
          aria-label={`Clear ${title} shortcut`}
          title="Clear shortcut"
        >
          <XIcon />
        </Button>
      {/if}
      <Button
        variant={capturing ? "default" : "secondary"}
        size="sm"
        onclick={capturing ? onCancel : onRecord}
        disabled={disabled}
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
  <div class="mt-2.5 min-h-5 text-xs font-medium" aria-live="polite" aria-atomic="true">
    {#if capturing}
      <p class="text-primary" role="status">
        Listening for this chord. Press Escape to cancel; captured keys will not trigger an action.
      </p>
    {:else if feedback}
      <p class={feedbackTone} role={feedback.state === "error" ? "alert" : "status"}>
        {feedback.message}
      </p>
    {/if}
  </div>
</div>
