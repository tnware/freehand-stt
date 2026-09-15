<script lang="ts">
  import { tick } from "svelte";
  import * as Popover from "$lib/components/ui/popover";
  import ChevronUpIcon from "@lucide/svelte/icons/chevron-up";
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  import TaskConnectionPanel from "./TaskConnectionPanel.svelte";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import { Button } from "$lib/components/ui/button";
  import { elapsedSeconds, State, type Status } from "$lib/state";
  import { Purpose } from "$bindings/savedconnection";
  import { isCopyRequired } from "$lib/utils/status";
  import type {
    TaskConnectionDetails,
    TaskConnectionStatus,
  } from "$lib/utils/connection";

  let {
    dictation,
    now,
    toggleShortcut = "",
    connectionState,
    connectionDetails,
    disabled = false,
    onCheck,
    onEdit,
    onSettings,
    cleanup = "",
    delivery = "",
    commandHint = "Ctrl+K",
    version = "",
    aboutOpen = false,
    onAbout,
  }: {
    dictation: Status;
    now: number;
    toggleShortcut?: string;
    connectionState: TaskConnectionStatus;
    connectionDetails: TaskConnectionDetails;
    disabled?: boolean;
    onCheck: () => void;
    onEdit: () => void;
    onSettings: () => void;
    /** The cleanup model in force, empty when cleanup is off. */
    cleanup?: string;
    /** Where a finished transcript goes. */
    delivery?: string;
    commandHint?: string;
    version?: string;
    aboutOpen?: boolean;
    onAbout: () => void;
  } = $props();

  let open = $state(false);
  async function navigate(action: () => void) {
    open = false;
    await tick();
    action();
  }

  const clock = $derived.by(() => {
    const seconds = elapsedSeconds(dictation, now);
    const minutes = Math.floor(seconds / 60);
    return `${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
  });

  /*
   * One canonical home for capture state, present on every pane. The chain's
   * input stage owns the rich control; this owns the fact, so moving to
   * Settings or Text to speech never hides whether the recorder is live.
   */
  const capture = $derived.by(() => {
    if (isCopyRequired(dictation))
      return { label: "Copy required", live: false, tone: "warn" };
    switch (dictation.state) {
      case State.Recording:
        return { label: "Recording", live: true, tone: "record" };
      case State.Transcribing:
        return { label: "Transcribing", live: true, tone: "busy" };
      case State.PostProcessing:
        return { label: "Cleaning up", live: true, tone: "busy" };
      case State.Ready:
        return { label: "Checking insertion target", live: true, tone: "busy" };
      case State.Cancelling:
        return { label: "Cancelling", live: true, tone: "busy" };
      case State.Failed:
        return { label: "Dictation failed", live: false, tone: "bad" };
      default:
        return { label: "Ready", live: false, tone: "idle" };
    }
  });
</script>

<!-- Endpoint reachability is different from capture state; both are ambient,
     so both live here rather than being restated inside each pane. -->
<footer
  class="status-bar flex h-6 shrink-0 items-center justify-between gap-2 border-t px-3 text-[11px] leading-none"
  data-tone={capture.tone}
>
  <div class="flex min-w-0 flex-1 items-center gap-2.5">
    <span
      class="flex shrink-0 items-center gap-2 rounded-sm px-1.5 py-1 {capture.tone ===
      'record'
        ? 'bg-record-wash text-record-text'
        : ''}"
    >
      <span
        class="size-2 shrink-0 rounded-full {capture.tone === 'record'
          ? 'bg-record'
          : capture.tone === 'warn'
            ? 'bg-warning'
            : capture.tone === 'bad'
              ? 'bg-destructive'
              : capture.tone === 'busy'
                ? 'bg-primary'
                : 'border border-muted-foreground'}"
        aria-hidden="true"
      ></span>
      <span class="figure font-medium" role="status">{capture.label}</span>
      {#if dictation.state === State.Recording}
        <span class="figure font-medium" aria-label={`Recording time ${clock}`}
          >{clock}</span
        >
      {/if}
      {#if !capture.live && toggleShortcut}
        <span class="hidden min-[760px]:inline">
          <ShortcutKeys value={toggleShortcut} label="Recording shortcut" />
        </span>
      {/if}
    </span>

    <span class="h-3 w-px shrink-0 bg-border" aria-hidden="true"></span>

    <Popover.Root bind:open>
      <Popover.Trigger
        aria-label={`${connectionState.scope} connection status: ${connectionState.label}`}
        class="flex h-6 min-w-0 items-center gap-2 rounded-sm px-1.5 text-left transition-colors hover:bg-subtle-fill-hover aria-expanded:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
      >
        <span
          class="size-2 shrink-0 rounded-full {connectionState.dot}"
          aria-hidden="true"
        ></span>
        <span class="truncate text-secondary-foreground"
          >{connectionState.label}</span
        >
        {#if connectionState.detail}<span
            class="hidden truncate text-ink-quiet min-[640px]:inline"
            >{connectionState.detail}</span
          >{/if}
        <ChevronUpIcon class="size-3 shrink-0 text-muted-foreground" />
      </Popover.Trigger>
      <Popover.Content
        side="top"
        align="start"
        role="dialog"
        aria-label="Active connection"
        class="w-[440px] max-w-[calc(100vw-24px)]"
      >
        {#key `${connectionDetails.purpose}:${connectionDetails.selected?.id ?? ""}`}
          <TaskConnectionPanel
            details={connectionDetails}
            {disabled}
            {onCheck}
            onEdit={() => void navigate(onEdit)}
            onSettings={() => void navigate(onSettings)}
          />
        {/key}
      </Popover.Content>
    </Popover.Root>

    {#if connectionDetails.model}
      <span class="h-3 w-px shrink-0 bg-border" aria-hidden="true"></span>
      <span
        class="hidden max-w-48 truncate font-mono text-[10px] text-ink-quiet min-[900px]:inline"
        title={connectionDetails.model}>{connectionDetails.model}</span
      >
    {/if}
  </div>

  <div class="flex shrink-0 items-center gap-2.5">
    <span class="hidden items-center gap-2.5 min-[1100px]:flex">
      {#if connectionDetails.purpose !== Purpose.Speech}
        <span
          class="max-w-40 truncate"
          title={cleanup ? `Cleanup ${cleanup}` : "No cleanup"}
          >{cleanup ? `Cleanup ${cleanup}` : "No cleanup"}</span
        >
        <span class="h-3 w-px bg-border" aria-hidden="true"></span>
      {/if}
      <span>{delivery}</span>
      <span class="h-3 w-px bg-border" aria-hidden="true"></span>
      <span class="font-mono text-muted-foreground">{commandHint}</span>
      <span class="h-3 w-px bg-border" aria-hidden="true"></span>
    </span>
    {#if version}
      <span class="figure text-ink-quiet">{version}</span>
    {/if}
    <Button
      variant="ghost"
      size="xs"
      class="-mr-1 h-6 rounded-sm px-1.5 text-[11px] font-normal text-secondary-foreground hover:text-foreground focus-visible:ring-inset"
      onclick={onAbout}
      aria-label={aboutOpen ? "Focus About" : "Open About"}
      title={aboutOpen ? "Focus About" : "Open About"}
    >
      <CircleHelpIcon class="size-3" />
      About
    </Button>
  </div>
</footer>

<style>
  /* The bar carries the capture state as a surface, so a live recorder is
     visible from across the room without a second indicator anywhere else. */
  .status-bar {
    border-color: var(--hairline);
    background: var(--layer-fill);
  }
  .status-bar[data-tone="record"] {
    border-color: var(--record-edge);
    background: color-mix(in srgb, var(--record) 9%, var(--layer-fill));
  }
</style>
