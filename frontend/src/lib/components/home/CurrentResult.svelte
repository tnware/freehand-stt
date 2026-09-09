<script lang="ts">
  import TranscriptText from "$lib/components/common/TranscriptText.svelte";
  import { followTranscript } from "$lib/utils/transcriptScroll";
  import { onDestroy, type Snippet } from "svelte";
  import { CopyFeedback } from "$lib/utils/copyFeedback.svelte";
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
  const feedback = new CopyFeedback();
  onDestroy(() => feedback.dispose());
  let following = $state(true);
  let jump = $state(0);
  async function copy() {
    const snapshot = resultKey;
    await feedback.copy(snapshot, onCopy);
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
        >{feedback.key === resultKey ? "Copied" : "Copy"}</Button
      >
    {/if}
  </div>
  <div class="flex min-h-0 flex-1 flex-col">
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
        {#if live || text}
          <div class="mx-auto w-full max-w-[76ch] p-4">
            {#if live}
              <p class="mb-3 text-xs text-muted-foreground" role="status">
                Live preview · text may change
              </p>
            {/if}
            <TranscriptText
              content={{
                key: resultKey,
                text: live ? liveFinal : text,
                partial: live
                  ? `${liveFinal && livePartial ? " " : ""}${livePartial || (!liveFinal ? "Listening…" : "")}`
                  : undefined,
              }}
              label={live ? "Live transcript" : "Current transcript"}
              class="whitespace-pre-wrap break-words text-sm leading-7"
            />
          </div>
        {:else if !message}
          <div class="flex flex-1 flex-col items-center justify-center gap-2 px-6 py-4 text-center">
            <p class="text-sm font-medium">
              {failed
                ? "No transcript to show"
                : working
                  ? "Your result will appear here"
                  : mode === "file"
                    ? "Turn an audio file into text"
                    : "Speak into the application you’re using"}
            </p>
            <p class="max-w-lg text-[13px] leading-relaxed text-muted-foreground">
              {failed
                ? mode === "file"
                  ? "Use Retry above, or choose another file."
                  : "Use Record again above when you’re ready."
                : working
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
      <div class="flex h-11 shrink-0 items-center justify-center border-t border-hairline px-3">
        <Button variant="ghost" size="sm" onclick={() => jump++}>Jump to latest</Button>
      </div>
    {/if}
  </div>
</section>
