<script lang="ts">
  import { onMount } from "svelte";
  import { Events, Window } from "@wailsio/runtime";
  import MinusIcon from "@lucide/svelte/icons/minus";
  import SquareIcon from "@lucide/svelte/icons/square";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import XIcon from "@lucide/svelte/icons/x";

  let { onWindowError }: { onWindowError?: (cause: unknown) => void } =
    $props();
  let maximized = $state(false);
  let mounted = false;
  let refreshState = () => {};

  onMount(() => {
    mounted = true;
    let revision = 0;
    let reading = false;
    let readFailureReported = false;

    // Keep at most one native read in flight. A later event invalidates an older
    // answer, including a resize emitted while a maximize request is settling.
    async function refresh() {
      revision += 1;
      if (reading) return;
      reading = true;
      try {
        while (mounted) {
          const requestedRevision = revision;
          try {
            const nextMaximized = await Window.IsMaximised();
            if (!mounted) return;
            if (requestedRevision === revision) {
              maximized = nextMaximized;
              readFailureReported = false;
            }
          } catch (cause) {
            if (mounted && !readFailureReported) {
              readFailureReported = true;
              onWindowError?.(cause);
            }
          }
          if (requestedRevision === revision) return;
        }
      } finally {
        reading = false;
      }
    }

    refreshState = () => void refresh();
    const subscriptions = [
      Events.Types.Common.WindowMaximise,
      Events.Types.Common.WindowUnMaximise,
      Events.Types.Common.WindowRestore,
      Events.Types.Common.WindowDidResize,
      Events.Types.Common.WindowShow,
      Events.Types.Common.WindowFocus,
    ].map((event) => Events.On(event, refreshState));
    refreshState();

    return () => {
      mounted = false;
      revision += 1;
      refreshState = () => {};
      for (const unsubscribe of subscriptions) unsubscribe();
    };
  });

  async function act(action: () => Promise<void>) {
    try {
      await action();
      if (mounted) refreshState();
    } catch (cause) {
      if (mounted) onWindowError?.(cause);
    }
  }
</script>

<div class="window-controls" role="group" aria-label="Window controls">
  <button
    type="button"
    class="caption-button minimize"
    aria-label="Minimize window"
    title="Minimize"
    onclick={() => act(() => Window.Minimise())}
  >
    <MinusIcon size={12} strokeWidth={1.25} aria-hidden="true" />
  </button>
  <button
    type="button"
    class="caption-button maximize"
    aria-label={maximized ? "Restore window" : "Maximize window"}
    title={maximized ? "Restore" : "Maximize"}
    onclick={() => act(() => Window.ToggleMaximise())}
  >
    {#if maximized}
      <CopyIcon size={12} strokeWidth={1.25} aria-hidden="true" />
    {:else}
      <SquareIcon size={11} strokeWidth={1.25} aria-hidden="true" />
    {/if}
  </button>
  <button
    type="button"
    class="caption-button close"
    aria-label="Close window"
    title="Close"
    onclick={() => act(() => Window.Close())}
  >
    <XIcon size={15} strokeWidth={1.25} aria-hidden="true" />
  </button>
</div>

<style>
  .window-controls {
    --wails-draggable: no-drag;
    display: flex;
    height: 100%;
    flex: none;
    align-items: stretch;
    margin-left: 6px;
  }

  .caption-button {
    display: grid;
    width: 46px;
    height: 100%;
    place-items: center;
    color: var(--foreground);
  }

  .minimize {
    --wails-non-client-region: minimize;
  }

  .maximize {
    --wails-non-client-region: maximize;
  }

  .close {
    --wails-non-client-region: close;
  }

  .caption-button:hover {
    background: var(--subtle-fill-hover);
  }

  .caption-button:active {
    background: var(--subtle-fill-pressed, var(--accent-wash));
  }

  .close:hover,
  .close:active {
    color: white;
    background: #c42b1c;
  }

  .caption-button:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -3px;
  }

  @media (forced-colors: active) {
    .caption-button:hover,
    .caption-button:active {
      background: Highlight;
      color: HighlightText;
      forced-color-adjust: none;
    }
  }
</style>
