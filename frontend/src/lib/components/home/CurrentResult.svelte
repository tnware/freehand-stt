<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  let {
    resultKey,
    mode,
    text,
    working,
    canCopy,
    recovery = false,
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
  class="flex min-h-56 flex-1 flex-col rounded-lg border border-hairline bg-layer-fill"
  aria-label="Current result"
>
  <div class="flex flex-wrap items-center gap-3 border-b border-hairline px-4 py-3">
    <h2 class="text-sm font-medium">Current result</h2>
    <span class="mr-auto text-xs text-muted-foreground"
      >{working
        ? "In progress"
        : recovery
          ? "Ready to copy"
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
  {#if text}
    {#if recovery}<p class="px-4 pt-3 text-xs text-warning">
        Freehand kept this result because it could not insert it. Copy it when you’re ready.
      </p>{/if}
    <div
      class="max-h-[45vh] overflow-y-auto whitespace-pre-wrap break-words p-4 text-sm leading-relaxed"
      tabindex="0"
      role="textbox"
      aria-readonly="true"
      aria-multiline="true"
      aria-label="Current transcript"
    >
      {text}
    </div>
  {:else}
    <div class="flex flex-1 flex-col items-center justify-center gap-2 p-8 text-center">
      <p class="text-sm font-medium">
        {working
          ? "Your result will appear here"
          : mode === "file"
            ? "Turn an audio file into text"
            : "Speak into the application you’re using"}
      </p>
      <p class="max-w-md text-xs leading-relaxed text-muted-foreground">
        {mode === "file"
          ? "Choose a file above. The transcript stays available here for inspection and copying."
          : "Use your recording shortcut from any application. Your latest transcript will also appear here."}
      </p>
    </div>
  {/if}
</section>
