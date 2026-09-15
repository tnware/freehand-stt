<script lang="ts">
  import SearchIcon from "@lucide/svelte/icons/search";
  import PanelLeftIcon from "@lucide/svelte/icons/panel-left";
  import PanelBottomIcon from "@lucide/svelte/icons/panel-bottom";
  import PanelRightIcon from "@lucide/svelte/icons/panel-right";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import * as Kbd from "$lib/components/ui/kbd";

  let {
    paneLabel,
    onOpenCommands,
    commandHint = "Ctrl",
    primaryVisible = true,
    bottomVisible = true,
    secondaryVisible = false,
    secondaryAvailable = false,
    bottomAvailable = true,
    onTogglePrimary,
    onToggleBottom,
    onToggleSecondary,
  }: {
    paneLabel: string;
    onOpenCommands: () => void;
    /** The platform's command modifier, so macOS does not read "Ctrl". */
    commandHint?: string;
    primaryVisible?: boolean;
    bottomVisible?: boolean;
    secondaryVisible?: boolean;
    secondaryAvailable?: boolean;
    bottomAvailable?: boolean;
    onTogglePrimary?: () => void;
    onToggleBottom?: () => void;
    onToggleSecondary?: () => void;
  } = $props();
</script>

<!--
  Identity, the current place, and one way into everything. The mode tabs this
  row used to carry are the activity rail, the shortcut and endpoint health are
  the status bar, and settings is the foot of the rail — which takes the header
  from 72px to 36px and gives the height back to the transcript.
-->
<header
  class="flex h-[34px] shrink-0 items-center gap-2.5 border-b border-hairline bg-well px-3"
>
  <div class="flex min-w-0 flex-1 items-center gap-2.5">
    <BrandMark />
    <h1 class="font-display text-[13px] font-semibold tracking-tight">
      Freehand
    </h1>
    <span class="h-3.5 w-px shrink-0 bg-border" aria-hidden="true"></span>
    <p class="truncate text-xs text-muted-foreground">{paneLabel}</p>
  </div>

  <button
    type="button"
    class="hidden h-[22px] w-[352px] shrink-0 items-center gap-2 rounded-md border border-border bg-subtle-fill-hover px-2.5 text-left transition-colors hover:border-accent-edge hover:bg-accent-wash focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring min-[760px]:flex"
    onclick={onOpenCommands}
  >
    <SearchIcon
      class="size-3 shrink-0 text-muted-foreground"
      aria-hidden="true"
    />
    <span class="flex-1 truncate text-[11.5px] text-ink-quiet"
      >Run a command or search settings</span
    >
    <Kbd.Group aria-hidden="true">
      <Kbd.Root>{commandHint}</Kbd.Root>
      <Kbd.Root>K</Kbd.Root>
    </Kbd.Group>
  </button>

  <div class="flex min-w-0 flex-1 items-center justify-end gap-0.5">
    <button
      type="button"
      class="grid size-6 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-subtle-fill-hover hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring min-[760px]:hidden"
      onclick={onOpenCommands}
      aria-label="Run a command"
      title="Run a command"
    >
      <SearchIcon class="size-3.5" aria-hidden="true" />
    </button>
    <div
      class="flex items-center gap-0.5"
      role="group"
      aria-label="Workspace layout"
    >
      <button
        type="button"
        class="layout-toggle"
        aria-label="Toggle primary sidebar"
        aria-controls="workbench-primary-sidebar"
        aria-pressed={primaryVisible}
        title={primaryVisible ? "Hide primary sidebar" : "Show primary sidebar"}
        disabled={!onTogglePrimary}
        onclick={onTogglePrimary}
        ><PanelLeftIcon class="size-3.5" aria-hidden="true" /></button
      >
      <button
        type="button"
        class="layout-toggle"
        aria-label="Toggle bottom panel"
        aria-controls="workbench-bottom-panel"
        aria-pressed={bottomVisible}
        title={!bottomAvailable
          ? "Increase window height to show the bottom panel"
          : bottomVisible
            ? "Hide bottom panel"
            : "Show bottom panel"}
        disabled={!bottomAvailable || !onToggleBottom}
        onclick={onToggleBottom}
        ><PanelBottomIcon class="size-3.5" aria-hidden="true" /></button
      >
      <button
        type="button"
        class="layout-toggle"
        aria-label="Toggle secondary sidebar"
        aria-controls="workbench-secondary-sidebar"
        aria-pressed={secondaryVisible}
        title={secondaryVisible
          ? "Hide secondary sidebar"
          : "Show secondary sidebar"}
        disabled={!secondaryAvailable || !onToggleSecondary}
        onclick={onToggleSecondary}
        ><PanelRightIcon class="size-3.5" aria-hidden="true" /></button
      >
    </div>
  </div>
</header>

<style>
  .layout-toggle {
    display: grid;
    width: 24px;
    height: 24px;
    place-items: center;
    border-radius: 4px;
    color: var(--muted-foreground);
    transition:
      color 120ms,
      background-color 120ms;
  }
  .layout-toggle[aria-pressed="true"] {
    background: var(--accent-wash);
    color: var(--accent-text);
  }
  .layout-toggle:hover:enabled {
    background: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  .layout-toggle:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }
  .layout-toggle:disabled {
    cursor: not-allowed;
    opacity: 0.4;
  }
  @media (prefers-reduced-motion: reduce) {
    .layout-toggle {
      transition: none;
    }
  }
</style>
