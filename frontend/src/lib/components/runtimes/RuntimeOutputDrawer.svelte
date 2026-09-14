<script lang="ts">
  import { onDestroy } from "svelte";
  import { Clipboard } from "@wailsio/runtime";
  import * as Manager from "$bindings/managedruntime/manager";
  import { Button } from "$lib/components/ui/button";
  import ProcessOutputTerminal from "$lib/components/ProcessOutputTerminal.svelte";
  import { ProcessOutputState } from "$lib/stores/process-output.svelte";

  let {
    instanceID = "",
    running = false,
  }: {
    /** The instance whose output to stream; empty releases the reader. */
    instanceID?: string;
    running?: boolean;
  } = $props();

  const output = new ProcessOutputState(Manager);
  let following = $state(true);

  // The reader is per instance and bounded to memory, so it follows the
  // selection and is released as soon as this drawer goes away.
  $effect(() => {
    const id = running ? instanceID : "";
    if (output.instanceID !== id) void output.select(id);
  });
  $effect(() => {
    if (!output.accepted) return;
    const timer = setInterval(() => void output.poll(), 1000);
    return () => clearInterval(timer);
  });
  onDestroy(() => output.dispose());
</script>

<div class="relative min-h-0 flex-1">
  {#if !running || !instanceID}
    <p
      class="flex h-full items-center justify-center px-4 text-center text-[12px] text-muted-foreground"
    >
      Output is available while a local runtime is running.
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
    >
      {#if !output.accepted}
        <!-- Output can include transcripts, prompts and file paths, so nothing
             streams until it is asked for. -->
        <div
          class="absolute inset-x-0 top-0 z-10 flex flex-wrap items-center justify-between gap-3 border-b border-warning/25 bg-card px-4 py-3"
        >
          <p
            id="runtime-output-consent"
            class="text-[12px] leading-relaxed text-secondary-foreground"
          >
            <span class="block font-semibold text-foreground"
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
    </ProcessOutputTerminal>
  {/if}
  {#if output.error}
    <p class="absolute inset-x-0 bottom-0 px-4 py-1 text-[11px] text-destructive" role="alert">
      {output.error}
    </p>
  {/if}
</div>
