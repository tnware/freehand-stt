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
        fontSize: 12,
        lineHeight: 1.25,
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
    void document.fonts.ready.then(() => {
      if (mounted) resize();
    });
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
    const snapshot = { chunks, revision, enabled, renderer };
    untrack(() =>
      snapshot.renderer?.sync(
        snapshot.chunks,
        snapshot.revision,
        snapshot.enabled,
      ),
    );
  });
</script>

<div class="flex min-h-0 flex-1 flex-col gap-3">
  <div
    class="flex shrink-0 flex-wrap items-center gap-1"
    role="group"
    aria-label="Output controls"
  >
    <Input
      class="h-8 w-44"
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
    <Button
      variant="ghost"
      size="sm"
      disabled={!enabled || !query}
      onclick={() => find(true)}>Previous</Button
    >
    <Button
      variant="ghost"
      size="sm"
      disabled={!enabled || !query}
      onclick={() => find()}>Next</Button
    >
    <Button
      variant="ghost"
      size="sm"
      disabled={!enabled}
      aria-pressed={following}
      onclick={() => {
        following = !following;
        if (following) terminal?.scrollToBottom();
      }}>Follow</Button
    >
    <Button
      variant="ghost"
      size="sm"
      disabled={!enabled || !selected}
      onclick={() => void copySelection()}>Copy selection</Button
    >
    <Button
      variant="ghost"
      size="sm"
      disabled={!enabled || busy}
      onclick={onclear}>Clear</Button
    >
    <span class="truncate text-xs text-muted-foreground" role="status"
      >{feedback}</span
    >
  </div>
  <div class="relative min-h-0 flex-1 border border-hairline bg-background">
    <div
      bind:this={viewport}
      role="region"
      aria-label="Read-only process output"
      aria-describedby={describedby}
      class="output-terminal absolute inset-0 overflow-hidden p-3"
    ></div>
    {@render children?.()}
  </div>
</div>

<style>
  .output-terminal :global(.xterm) {
    height: 100%;
  }
</style>
