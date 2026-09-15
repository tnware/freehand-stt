<script lang="ts">
  import { onMount, untrack, type Snippet } from "svelte";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import { SearchAddon } from "@xterm/addon-search";
  import "@xterm/xterm/css/xterm.css";
  import "@fontsource/ibm-plex-mono/400.css";
  import type { OutputChunk } from "$bindings/managedruntime/models";
  import { ProcessOutputRenderer } from "$lib/process-output-renderer";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import ArrowDownToLineIcon from "@lucide/svelte/icons/arrow-down-to-line";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import ArrowUpIcon from "@lucide/svelte/icons/arrow-up";
  import ArrowDownIcon from "@lucide/svelte/icons/arrow-down";
  import SearchIcon from "@lucide/svelte/icons/search";
  import EraserIcon from "@lucide/svelte/icons/eraser";
  import HighlighterIcon from "@lucide/svelte/icons/highlighter";

  let {
    chunks,
    revision,
    enabled,
    busy = false,
    following = $bindable(true),
    onclear,
    oncopy,
    describedby,
    children,
  }: {
    chunks: readonly OutputChunk[];
    revision: number;
    enabled: boolean;
    busy?: boolean;
    following?: boolean;
    onclear: () => void;
    oncopy: (selection: string) => Promise<void>;
    describedby?: string;
    children?: Snippet;
  } = $props();

  let viewport: HTMLDivElement | undefined = $state();
  let renderer = $state.raw<ProcessOutputRenderer>();
  let selected = $state(false);
  let query = $state("");
  let feedback = $state("");
  let fontReady = $state(false);
  let highlight = $state(true);
  let terminal: Terminal | undefined;
  let search: SearchAddon | undefined;
  let fit: FitAddon | undefined;

  function updateTheme() {
    if (!terminal) return;
    const styles = getComputedStyle(document.documentElement);
    terminal.options.theme = {
      background: styles.getPropertyValue("--background").trim(),
      foreground: styles.getPropertyValue("--foreground").trim(),
      selectionBackground: styles.getPropertyValue("--primary").trim(),
      selectionInactiveBackground: styles.getPropertyValue("--primary").trim(),
      selectionForeground: styles
        .getPropertyValue("--primary-foreground")
        .trim(),
      // A transparent bar hides the caret without hiding the cell beneath it.
      cursor: "#00000000",
      red: styles.getPropertyValue("--destructive").trim(),
      brightRed: styles.getPropertyValue("--destructive").trim(),
      green: styles.getPropertyValue("--success").trim(),
      brightGreen: styles.getPropertyValue("--success").trim(),
      yellow: styles.getPropertyValue("--warning").trim(),
      brightYellow: styles.getPropertyValue("--warning").trim(),
      blue: styles.getPropertyValue("--accent-text").trim(),
      brightBlue: styles.getPropertyValue("--accent-text").trim(),
    };
  }
  function resize() {
    if (!viewport?.clientWidth || !viewport.clientHeight) return;
    fit?.fit();
    if (following) terminal?.scrollToBottom();
  }
  function find(previous = false) {
    if (!enabled || !terminal || !search) return;
    feedback = "";
    if (!query) {
      search.clearDecorations();
      terminal.clearSelection();
      return;
    }
    following = false;
    const found = previous
      ? search.findPrevious(query, { regex: false })
      : search.findNext(query, { regex: false });
    // Search can clear and reselect the same match without a second selection
    // event. Read the final selection so Copy reflects the highlighted text.
    selected = terminal.hasSelection();
    feedback = found ? "" : "No match in retained output.";
  }
  async function copySelection() {
    if (!enabled || !terminal) return;
    const current = terminal;
    const selection = current.getSelection();
    if (!selection) return;
    try {
      await oncopy(selection);
      if (terminal === current) feedback = "Selection copied.";
    } catch {
      if (terminal === current) feedback = "Could not copy selection.";
    }
  }

  onMount(() => {
    if (!viewport) return;
    const container = viewport;
    const projection = new ProcessOutputRenderer(() => {
      // The backend admits only bounded SGR/erase-line and CR/BS/HT/LF.
      // No stdin, PTY, events, title, links or clipboard escape integrations.
      const current = new Terminal({
        disableStdin: true,
        convertEol: true,
        scrollback: 4096,
        reflowCursorLine: true,
        fontFamily: '"IBM Plex Mono", monospace',
        fontSize: 13,
        lineHeight: 1.4,
        cursorBlink: false,
        cursorStyle: "bar",
        cursorInactiveStyle: "none",
        altClickMovesCursor: false,
        scrollOnUserInput: false,
        screenReaderMode: true,
        minimumContrastRatio: 4.5,
        logLevel: "off",
      });
      terminal = current;
      fit = new FitAddon();
      search = new SearchAddon();
      current.loadAddon(fit);
      current.loadAddon(search);
      current.open(container);
      // Keep the selection surface keyboard-accessible, but never editable.
      if (current.textarea) {
        current.textarea.readOnly = true;
        current.textarea.setAttribute("aria-label", "Read-only process output");
      }
      updateTheme();
      resize();
      const selection = current.onSelectionChange(() => {
        selected = current.hasSelection();
      });
      return {
        write(text, done) {
          const top = current.buffer.active.viewportY;
          current.write(text, () => {
            if (terminal === current) {
              if (following) current.scrollToBottom();
              else current.scrollToLine(top);
            }
            done();
          });
        },
        dispose() {
          terminal = undefined;
          fit = undefined;
          search = undefined;
          selected = false;
          query = "";
          feedback = "";
          selection.dispose();
          // reset()/clear() cannot cancel xterm's already queued parser work.
          current.dispose();
          container.replaceChildren();
        },
      };
    });
    renderer = projection;
    const observer = new ResizeObserver(resize);
    observer.observe(container);
    const appearance = new MutationObserver(updateTheme);
    appearance.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "style", "data-material"],
    });
    let mounted = true;
    function finishFontLoad() {
      if (!mounted) return;
      fontReady = true;
      resize();
    }
    // An unused webfont is not covered by fonts.ready. Load it before xterm
    // caches glyph widths; otherwise a late font swap can clip wrapped text.
    void document.fonts
      .load('13px "IBM Plex Mono"')
      .then(finishFontLoad, finishFontLoad);
    return () => {
      mounted = false;
      observer.disconnect();
      appearance.disconnect();
      projection.dispose();
    };
  });

  $effect.pre(() => {
    // Imperative xterm lifecycle, not derived UI state. Do not subscribe to
    // incidental reads of selection/follow/theme while creating the terminal.
    const snapshot = {
      chunks,
      revision,
      enabled: enabled && fontReady,
      highlight,
      renderer,
    };
    untrack(() =>
      snapshot.renderer?.sync(
        snapshot.chunks,
        snapshot.revision,
        snapshot.enabled,
        snapshot.highlight,
      ),
    );
  });
</script>

<div class="flex min-h-0 min-w-0 flex-1 flex-col">
  <div
    class="flex min-w-0 shrink-0 flex-wrap items-center gap-x-2 gap-y-1 border-b border-hairline px-3 py-1"
    role="group"
    aria-label="Output controls"
  >
    <div class="flex min-w-0 flex-1 items-center gap-0.5">
      <div class="relative min-w-0 max-w-64 flex-1">
        <SearchIcon
          class="pointer-events-none absolute top-1/2 left-2 size-3 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          class="h-6 min-w-0 rounded-sm pl-7 text-xs"
          type="search"
          aria-label="Search retained output"
          placeholder="Search output"
          maxlength={256}
          disabled={!enabled}
          bind:value={query}
          oninput={() => (feedback = "")}
          onkeydown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              find(event.shiftKey);
            }
          }}
        />
      </div>
      <Button
        variant="ghost"
        size="icon-xs"
        class="rounded-sm"
        aria-label="Previous"
        title="Previous match (Shift+Enter)"
        disabled={!enabled || !query}
        onclick={() => find(true)}
        ><ArrowUpIcon class="size-3.5" aria-hidden="true" /></Button
      >
      <Button
        variant="ghost"
        size="icon-xs"
        class="rounded-sm"
        aria-label="Next"
        title="Next match (Enter)"
        disabled={!enabled || !query}
        onclick={() => find()}
        ><ArrowDownIcon class="size-3.5" aria-hidden="true" /></Button
      >
    </div>
    <div class="flex shrink-0 items-center gap-0.5">
      <Button
        variant="ghost"
        size="icon-xs"
        class={highlight
          ? "rounded-sm bg-accent-wash text-accent-text"
          : "rounded-sm"}
        aria-label="Highlight logs"
        title="Highlight recognized log events and severity"
        disabled={!enabled}
        aria-pressed={highlight}
        onclick={() => (highlight = !highlight)}
        ><HighlighterIcon class="size-3.5" aria-hidden="true" /></Button
      >
      <Button
        variant="ghost"
        size="icon-xs"
        class={following
          ? "rounded-sm bg-accent-wash text-accent-text"
          : "rounded-sm"}
        aria-label="Follow"
        title={following ? "Stop following new output" : "Follow new output"}
        disabled={!enabled}
        aria-pressed={following}
        onclick={() => {
          following = !following;
          if (following) terminal?.scrollToBottom();
        }}><ArrowDownToLineIcon class="size-3.5" aria-hidden="true" /></Button
      >
      <Button
        variant="ghost"
        size="icon-xs"
        class="rounded-sm"
        aria-label="Copy selection"
        title="Copy selection"
        disabled={!enabled || !selected}
        onclick={() => void copySelection()}
        ><CopyIcon class="size-3.5" aria-hidden="true" /></Button
      >
      <Button
        variant="ghost"
        size="icon-xs"
        class="rounded-sm"
        aria-label="Clear"
        title="Clear retained output"
        disabled={!enabled || busy}
        onclick={onclear}
        ><EraserIcon class="size-3.5" aria-hidden="true" /></Button
      >
    </div>
    <span
      class="w-full text-xs text-muted-foreground empty:hidden"
      role="status">{feedback}</span
    >
  </div>
  <div class="relative min-h-0 flex-1 overflow-hidden bg-background">
    <div
      bind:this={viewport}
      role="region"
      aria-label="Read-only process output"
      aria-describedby={describedby}
      class="output-terminal absolute inset-x-3 inset-y-2 overflow-hidden"
    ></div>
    {@render children?.()}
  </div>
</div>

<style>
  .output-terminal :global(.xterm) {
    height: 100%;
  }
  .output-terminal :global(.xterm-viewport) {
    background-color: var(--background);
  }
</style>
