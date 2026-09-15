<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { Clipboard, Events } from "@wailsio/runtime";
  import * as Manager from "$bindings/managedruntime/manager";
  import { Button } from "$lib/components/ui/button";
  import ProcessOutputTerminal from "$lib/components/ProcessOutputTerminal.svelte";
  import { ProcessOutputState } from "$lib/stores/process-output.svelte";

  let {
    instanceID = "",
  }: {
    /** The instance whose output to stream; empty releases the reader. */
    instanceID?: string;
  } = $props();

  const output = new ProcessOutputState(Manager);
  let following = $state(true);

  // The reader is per instance and bounded to memory, so it follows the
  // selection and is released as soon as this drawer goes away.
  $effect(() => {
    const id = instanceID;
    if (output.instanceID !== id) void output.select(id);
  });
  $effect(() => {
    if (!output.accepted) return;
    const timer = setInterval(() => void output.poll(), 1000);
    return () => clearInterval(timer);
  });
  onDestroy(() => output.dispose());
  onMount(() => {
    const clear = () => {
      following = true;
      void output.select(instanceID);
    };
    const off = Events.On("shell:hidden", clear);
    const visibilityChanged = () => {
      if (document.hidden) clear();
    };
    document.addEventListener("visibilitychange", visibilityChanged);
    return () => {
      off();
      document.removeEventListener("visibilitychange", visibilityChanged);
    };
  });
</script>

<div class="relative flex min-h-0 flex-1 flex-col">
  {#if !instanceID}
    <p
      class="flex h-full items-center justify-center px-5 text-center text-[12px] text-muted-foreground"
    >
      Install a local runtime to inspect its output.
    </p>
  {:else}
    <ProcessOutputTerminal
      chunks={output.chunks}
      revision={output.revision}
      enabled={output.accepted}
      busy={output.busy}
      bind:following
      onclear={() => void output.clear()}
      oncopy={Clipboard.SetText}
      describedby={!output.accepted ? "runtime-output-consent" : undefined}
    />
    {#if !output.accepted}
      <!-- Output can include transcripts, prompts and file paths, so nothing
             streams until it is asked for. -->
      <div
        class="absolute inset-0 z-10 flex flex-wrap items-center justify-between gap-3 overflow-y-auto bg-background px-5 py-3"
      >
        <p
          id="runtime-output-consent"
          class="min-w-0 flex-1 basis-64 text-[12px] leading-relaxed text-secondary-foreground"
        >
          <span class="block text-[13px] font-semibold text-foreground"
            >Show sensitive output</span
          >
          Output may include transcripts, prompts and file paths.
        </p>
        <Button
          size="xs"
          class="shrink-0"
          disabled={output.busy}
          onclick={() => void output.show()}>Show output</Button
        >
      </div>
    {/if}
  {/if}
  {#if output.error}
    <p
      class="absolute inset-x-0 bottom-0 z-20 bg-background px-5 py-1 text-xs text-destructive"
      role="alert"
    >
      {output.error}
    </p>
  {/if}
</div>
