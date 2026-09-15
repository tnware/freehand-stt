<script lang="ts">
  import { onMount, type Snippet } from "svelte";
  import { System } from "@wailsio/runtime";
  import SearchIcon from "@lucide/svelte/icons/search";
  import PanelLeftIcon from "@lucide/svelte/icons/panel-left";
  import PanelBottomIcon from "@lucide/svelte/icons/panel-bottom";
  import PanelRightIcon from "@lucide/svelte/icons/panel-right";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import WindowControls from "$lib/components/shell/WindowControls.svelte";
  import * as Kbd from "$lib/components/ui/kbd";

  let {
    onOpenCommands,
    commandHint = "Ctrl",
    primaryVisible = true,
    bottomVisible = true,
    secondaryVisible = false,
    secondaryAvailable = false,
    bottomAvailable = true,
    bottomUnavailableReason = "Increase window height to show the bottom panel",
    onTogglePrimary,
    onToggleBottom,
    onToggleSecondary,
    platform,
    onWindowError,
    menu,
  }: {
    onOpenCommands: () => void;
    /** The platform's command modifier, so macOS does not read "Ctrl". */
    commandHint?: string;
    primaryVisible?: boolean;
    bottomVisible?: boolean;
    secondaryVisible?: boolean;
    secondaryAvailable?: boolean;
    bottomAvailable?: boolean;
    bottomUnavailableReason?: string;
    onTogglePrimary?: () => void;
    onToggleBottom?: () => void;
    onToggleSecondary?: () => void;
    platform?: string;
    onWindowError?: (cause: unknown) => void;
    menu?: Snippet;
  } = $props();

  let nativePlatform = $state<string>();
  const activePlatform = $derived(platform ?? nativePlatform);

  onMount(() => {
    function readPlatform() {
      nativePlatform = System.IsMac()
        ? "darwin"
        : System.IsWindows()
          ? "windows"
          : undefined;
    }
    readPlatform();
    window.addEventListener("wails:runtime-config-ready", readPlatform);
    return () =>
      window.removeEventListener("wails:runtime-config-ready", readPlatform);
  });
</script>

<!-- Caption rectangles must not span any interactive descendants. -->
<header
  class="title-bar border-b border-hairline bg-well"
  class:windows={activePlatform === "windows"}
  class:mac={activePlatform === "darwin"}
>
  <div class="title-identity">
    <BrandMark />
    <h1 class="font-display text-[13px] font-semibold tracking-tight">
      Freehand
    </h1>
  </div>
  {#if menu && activePlatform === "windows"}
    <div class="title-menu">{@render menu()}</div>
  {/if}
  <div class="title-drag-space" aria-hidden="true"></div>

  <button
    type="button"
    class="command-search h-[24px] items-center gap-2 rounded-sm border border-border bg-subtle-fill-hover px-2.5 text-left transition-colors hover:border-accent-edge hover:bg-accent-wash focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring"
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

  <div class="title-drag-space" aria-hidden="true"></div>
  <div class="title-actions flex shrink-0 items-center justify-end gap-0.5">
    <button
      type="button"
      class="compact-search grid size-6 place-items-center rounded-sm text-muted-foreground transition-colors hover:bg-subtle-fill-hover hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
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
          ? bottomUnavailableReason
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
  {#if activePlatform === "windows"}
    <WindowControls {onWindowError} />
  {/if}
</header>

<style>
  .title-bar {
    --wails-draggable: no-drag;
    display: flex;
    height: 36px;
    flex: none;
    align-items: center;
    gap: 6px;
    padding: 0 10px;
  }

  .title-bar.windows {
    padding-right: 0;
  }

  .title-bar.mac {
    height: 44px;
    padding-left: 80px;
  }

  .title-identity {
    display: flex;
    height: 100%;
    flex: none;
    align-items: center;
    gap: 8px;
    user-select: none;
  }

  .title-identity h1 {
    flex: none;
  }

  .title-drag-space {
    height: 100%;
    min-width: 8px;
    flex: 1;
  }

  .windows .title-identity,
  .windows .title-drag-space {
    --wails-non-client-region: caption;
  }

  .windows .title-identity,
  .windows .title-drag-space,
  .mac .title-identity,
  .mac .title-drag-space {
    --wails-draggable: drag;
  }

  .title-menu {
    --wails-draggable: no-drag;
    flex: none;
  }

  .command-search {
    display: none;
    width: clamp(240px, 28vw, 352px);
    flex: none;
  }

  @media (min-width: 960px) {
    .command-search {
      display: flex;
    }

    .compact-search {
      display: none;
    }
  }

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
