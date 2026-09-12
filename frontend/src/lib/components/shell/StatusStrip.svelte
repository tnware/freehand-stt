<script lang="ts">
  import { tick } from "svelte";
  import * as Popover from "$lib/components/ui/popover";
  import ChevronUpIcon from "@lucide/svelte/icons/chevron-up";
  import TaskConnectionPanel from "./TaskConnectionPanel.svelte";
  import type { TaskConnectionDetails } from "$lib/utils/connection";
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  import { Button } from "$lib/components/ui/button";
  import type { TaskConnectionStatus } from "$lib/utils/connection";

  let {
    connectionState,
    connectionDetails,
    disabled = false,
    onCheck,
    onEdit,
    onSettings,
    version = "",
    aboutOpen = false,
    onAbout,
  }: {
    connectionState: TaskConnectionStatus;
    connectionDetails: TaskConnectionDetails;
    disabled?: boolean;
    onCheck: () => void;
    onEdit: () => void;
    onSettings: () => void;
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
</script>

<!-- Endpoint reachability is different from dictation state. Keep it quiet,
     persistent, and truthful about whether a probe has actually run. -->
<footer
  class="flex h-7 shrink-0 items-center justify-between gap-2 border-t border-hairline bg-layer-fill px-3 text-[11px] leading-none"
>
  <Popover.Root bind:open>
    <Popover.Trigger
      aria-label={`${connectionState.scope} connection status: ${connectionState.label}`}
      class="-ml-2 flex min-w-0 items-center gap-1.5 h-6 rounded-md px-2 text-left transition-colors hover:bg-subtle-fill-hover aria-expanded:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring"
    >
      <span class="size-1.5 shrink-0 rounded-full {connectionState.dot}"></span>
      <span class="truncate text-secondary-foreground"
        >{connectionState.scope}: {connectionState.label}</span
      >
      {#if connectionState.detail}<span class="hidden truncate text-ink-quiet sm:inline"
          >{connectionState.detail}</span
        >{/if}
      <ChevronUpIcon class="size-3 shrink-0 text-muted-foreground" />
    </Popover.Trigger>
    <Popover.Content
      side="top"
      align="start"
      role="dialog"
      aria-label="Active connection"
      class="w-[440px]"
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
  <div class="flex shrink-0 items-center gap-3">
    {#if version}
      <span class="text-ink-quiet">{version}</span>
    {/if}
    <Button
      variant="ghost"
      size="xs"
      class="-mr-1 h-6 rounded-md px-1.5 text-[11px] font-normal text-secondary-foreground hover:text-foreground focus-visible:ring-inset"
      onclick={onAbout}
      aria-label={aboutOpen ? "Focus About" : "Open About"}
      title={aboutOpen ? "Focus About" : "Open About"}
    >
      <CircleHelpIcon class="size-3" />
      About
    </Button>
  </div>
</footer>
