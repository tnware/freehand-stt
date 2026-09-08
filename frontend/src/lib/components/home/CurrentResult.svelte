<script lang="ts">
  import { followTranscript } from "$lib/utils/transcriptScroll";
  import type { Snippet } from "svelte";
  import { Button } from "$lib/components/ui/button";
  let {
    live = false,
    liveFinal = "",
    livePartial = "",
    resultKey,
    mode,
    text,
    working,
    canCopy,
    recovery = false,
    message = "",
    failed = false,
    onCopy,
    onClear,
    onListen,
    quickSettings,
  }: {
    live?: boolean;
    liveFinal?: string;
    livePartial?: string;
    quickSettings?: Snippet;
    resultKey: string;
    mode: "voice" | "file";
    text: string;
    working: boolean;
    canCopy: boolean;
    recovery?: boolean;
    message?: string;
    failed?: boolean;
    onCopy: () => Promise<boolean>;
    onClear: () => void;
    onListen?: () => void;
  } = $props();
  let copiedKey = $state("");
  let following = $state(true);
  let jump = $state(0);
  async function copy() {
    const snapshot = resultKey;
    if (await onCopy()) copiedKey = snapshot;
  }
</script>

<section
  class="@container flex min-h-40 flex-1 flex-col overflow-hidden rounded-lg border border-hairline bg-layer-fill"
  aria-label="Current result"
>
  <div class="flex h-12 shrink-0 items-center gap-1 border-b border-hairline px-3">
    <h2 class={quickSettings ? "sr-only" : "text-sm font-medium"}>Current result</h2>
    <span class={quickSettings ? "sr-only" : "mr-auto text-xs text-muted-foreground"}
      >{working
        ? "In progress"
        : recovery
          ? "Ready to copy"
          : failed
            ? "Needs attention"
            : text
              ? "Ready"
              : mode === "file"
                ? "No transcript yet"
                : "Nothing recorded yet"}</span
    >
    {#if quickSettings}
      <div class="min-w-0 flex-1">{@render quickSettings()}</div>
    {/if}
    {#if text}
      {#if onListen}<Button
          variant="ghost"
          size="sm"
          disabled={working || !canCopy}
          onclick={onListen}>Listen</Button
        >{/if}
      <Button variant="ghost" size="sm" disabled={working} onclick={onClear}>Clear</Button>
      <Button variant="outline" size="sm" disabled={working || !canCopy} onclick={copy}
        >{copiedKey === resultKey ? "Copied" : "Copy"}</Button
      >
    {/if}
  </div>
  <div class="relative flex min-h-0 flex-1 flex-col">
    <div
      class="min-h-0 flex-1 overflow-y-auto overscroll-contain [overflow-anchor:none]"
      use:followTranscript={{
        key: resultKey,
        content: `${text}\n${liveFinal}\n${livePartial}`,
        streaming: live || working,
        jump,
        onFollowingChange: (value) => (following = value),
      }}
    >
      <div class="flex min-h-full flex-col">
        {#if message || recovery}
          <p
            class="px-4 pt-3 text-xs leading-relaxed"
            class:text-warning={recovery}
            class:text-destructive={failed && !recovery}
            role="status"
          >
            {message ||
              "Freehand kept this result because it could not insert it. Copy it when you’re ready."}
          </p>
        {/if}
        {#if live}
          <div class="mx-auto w-full max-w-[76ch] p-4 pb-14">
            <p class="mb-3 text-xs text-muted-foreground" role="status">
              Live preview · text may change
            </p>
            <div
              class="whitespace-pre-wrap break-words text-sm leading-7"
              aria-label="Live transcript"
            >
              {liveFinal}<span class="text-muted-foreground"
                >{liveFinal && livePartial ? " " : ""}{livePartial ||
                  (!liveFinal ? "Listening…" : "")}</span
              >
            </div>
          </div>
        {:else if text}
          <div
            class="mx-auto w-full max-w-[76ch] whitespace-pre-wrap break-words p-4 pb-14 text-sm leading-7"
            tabindex="0"
            role="textbox"
            aria-readonly="true"
            aria-multiline="true"
            aria-label="Current transcript"
          >
            {text}
          </div>
        {:else if !message}
          <div class="flex flex-1 flex-col items-center justify-center gap-2 px-6 py-4 text-center">
            <p class="text-sm font-medium">
              {working
                ? "Your result will appear here"
                : mode === "file"
                  ? "Turn an audio file into text"
                  : "Speak into the application you’re using"}
            </p>
            <p class="max-w-lg text-[13px] leading-relaxed text-muted-foreground">
              {working
                ? "You can keep working while Freehand finishes."
                : mode === "file"
                  ? "Choose a file above. The transcript stays available here for inspection and copying."
                  : "Use your recording shortcut from any application. Your latest transcript will also appear here."}
            </p>
          </div>
        {/if}
      </div>
    </div>
    {#if !following && (text || liveFinal || livePartial)}
      <div class="pointer-events-none absolute inset-x-0 bottom-3 flex justify-center">
        <Button
          variant="secondary"
          size="sm"
          class="pointer-events-auto border border-border shadow-md"
          onclick={() => jump++}>Jump to latest</Button
        >
      </div>
    {/if}
  </div>
</section>
