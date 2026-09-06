<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  let {
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
  }: {
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
  async function copy() {
    const snapshot = resultKey;
    if (await onCopy()) copiedKey = snapshot;
  }
</script>

<section
  class="flex h-52 shrink-0 flex-col overflow-hidden rounded-lg border border-hairline bg-layer-fill"
  aria-label="Current result"
>
  <div class="flex h-14 shrink-0 items-center gap-3 border-b border-hairline px-4 py-3">
    <h2 class="text-sm font-medium">Current result</h2>
    <span class="mr-auto text-xs text-muted-foreground"
      >{working
        ? "In progress"
        : recovery
          ? "Ready to copy"
          : failed
            ? "Needs attention"
            : text
              ? "Ready"
              : "Nothing recorded yet"}</span
    >
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
  <div class="min-h-0 flex-1 overflow-y-auto">
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
    {#if text}
      <div
        class="whitespace-pre-wrap break-words p-4 text-sm leading-relaxed"
        tabindex="0"
        role="textbox"
        aria-readonly="true"
        aria-multiline="true"
        aria-label="Current transcript"
      >
        {text}
      </div>
    {:else if !message}
      <div class="flex h-full flex-col items-center justify-center gap-2 px-6 py-4 text-center">
        <p class="text-sm font-medium">
          {working
            ? "Your result will appear here"
            : mode === "file"
              ? "Turn an audio file into text"
              : "Speak into the application you’re using"}
        </p>
        <p class="max-w-md text-xs leading-relaxed text-muted-foreground">
          {working
            ? "You can keep working while Freehand finishes."
            : mode === "file"
              ? "Choose a file above. The transcript stays available here for inspection and copying."
              : "Use your recording shortcut from any application. Your latest transcript will also appear here."}
        </p>
      </div>
    {/if}
  </div>
</section>
