<script lang="ts">
  import TranscriptText from "$lib/components/common/TranscriptText.svelte";
  import { followTranscript } from "$lib/utils/transcriptScroll";
  import { onDestroy } from "svelte";
  import { CopyFeedback } from "$lib/utils/copyFeedback.svelte";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import { Button } from "$lib/components/ui/button";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import AudioLinesIcon from "@lucide/svelte/icons/audio-lines";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
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
    listenBusy = false,
    listenDisabled = false,
  }: {
    live?: boolean;
    liveFinal?: string;
    livePartial?: string;
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
    listenBusy?: boolean;
    listenDisabled?: boolean;
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
  class="@container flex min-h-40 flex-1 flex-col overflow-hidden"
  aria-label="Current result"
>
  <div class="workbench-toolbar flex-wrap gap-y-1 py-1">
    <h2 class="content-title">Transcript</h2>
    <div class="mr-auto min-w-0" role="status">
      <StatusBadge
        tone={live || working
          ? "danger"
          : recovery
            ? "warning"
            : failed
              ? "danger"
              : "neutral"}
        class={live || working
          ? "border-record-edge bg-record-wash text-record-text"
          : text
            ? "font-semibold"
            : ""}
      >
        {#if live}<span
            class="size-1.5 shrink-0 rounded-full bg-record motion-safe:animate-pulse"
            aria-hidden="true"
          ></span>{/if}{live
          ? "Live"
          : working
            ? "In progress"
            : recovery
              ? "Ready to copy"
              : failed
                ? "Needs attention"
                : text
                  ? "Ready"
                  : mode === "file"
                    ? "No transcript yet"
                    : "Nothing recorded yet"}
      </StatusBadge>
    </div>
    {#if text}
      <div class="ml-auto flex shrink-0 items-center gap-1">
        {#if onListen}<Button
            variant="outline"
            size="xs"
            class="min-w-14"
            disabled={working || !canCopy || listenDisabled}
            aria-label={listenBusy
              ? "Preparing speech for this transcript"
              : "Listen"}
            aria-busy={listenBusy}
            title={listenBusy
              ? "Preparing speech for this transcript"
              : listenDisabled
                ? "Wait for speech generation to finish"
                : "Listen to transcript"}
            onclick={onListen}
            >{#if listenBusy}<LoaderCircleIcon
                class="animate-spin motion-reduce:animate-none"
              />{:else}Listen{/if}</Button
          >{/if}
        <Button variant="ghost" size="xs" disabled={working} onclick={onClear}
          >Clear</Button
        >
        <Button
          variant="soft"
          size="xs"
          class="min-w-16"
          disabled={working || !canCopy}
          onclick={copy}
          >{feedback.key === resultKey ? "Copied" : "Copy"}</Button
        >
      </div>
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
            class="border-b border-hairline bg-layer-fill px-3 py-2 text-xs leading-relaxed"
            class:text-warning={recovery}
            class:text-destructive={failed && !recovery}
            role="status"
          >
            {message ||
              "Freehand kept this result because it could not insert it. Copy it when you’re ready."}
          </p>
        {/if}
        {#if live || text}
          <div class="w-full max-w-[760px] px-3 py-3.5">
            {#if live}
              <p class="content-meta mb-3" role="status">
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
              class="whitespace-pre-wrap break-words text-[15px] leading-[26px] text-foreground"
            />
          </div>
        {:else if !message}
          <div
            class="flex flex-1 flex-col items-center justify-center gap-3 px-5 py-8 text-center"
          >
            <span
              class="grid size-12 place-items-center rounded-lg border border-hairline bg-well text-muted-foreground shadow-lift"
              aria-hidden="true"
            >
              {#if working}<LoaderCircleIcon
                  class="size-6 animate-spin text-accent-text motion-reduce:animate-none"
                />
              {:else if mode === "file"}<FileAudioIcon class="size-6" />
              {:else}<AudioLinesIcon class="size-6 text-accent-text" />{/if}
            </span>
            <p class="text-[17px] font-semibold text-foreground">
              {failed
                ? "No transcript to show"
                : working
                  ? "Your result will appear here"
                  : mode === "file"
                    ? "Turn an audio file into text"
                    : "Speak into the application you’re using"}
            </p>
            <p class="content-meta max-w-lg">
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
      <div
        class="flex min-h-8 shrink-0 items-center justify-center border-t border-hairline px-3"
      >
        <Button variant="ghost" size="sm" onclick={() => jump++}
          >Jump to latest</Button
        >
      </div>
    {/if}
  </div>
</section>
