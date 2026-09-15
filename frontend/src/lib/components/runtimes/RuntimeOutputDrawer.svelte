<script lang="ts">
  import { onDestroy, onMount, untrack } from "svelte";
  import { Clipboard, Events, Window } from "@wailsio/runtime";
  import * as Manager from "$bindings/managedruntime/manager";
  import { SettingsVisible } from "$bindings/windowing/service";
  import { Button } from "$lib/components/ui/button";
  import ProcessOutputTerminal from "$lib/components/ProcessOutputTerminal.svelte";
  import { ProcessOutputState } from "$lib/stores/process-output.svelte";

  let {
    instanceID = "",
  }: {
    /** The instance to inspect while this output tab is visible. */
    instanceID?: string;
  } = $props();

  const output = new ProcessOutputState(Manager);
  let following = $state(true);
  let mounted = false;
  let shellVisible = $state(false);
  let visibilityError = $state("");
  let readVisibility: () => Promise<void> = async () => {};

  // Selection and visibility are the only automatic reader triggers. A failed
  // enable waits for an explicit retry instead of repeatedly reopening.
  $effect(() => {
    const id = shellVisible ? instanceID : "";
    untrack(() => {
      output.visible = shellVisible;
      // Runtime status refreshes can invalidate the prop without changing its
      // target. Keep the active reader and scroll position in that case.
      if (output.instanceID === id) return;
      following = true;
      void output.open(id);
    });
  });
  $effect(() => {
    if (!output.accepted || !shellVisible) return;
    const timer = setInterval(() => void output.poll(), 1000);
    return () => clearInterval(timer);
  });
  onDestroy(() => output.dispose());
  onMount(() => {
    mounted = true;
    let alive = true;
    let revision = 0;
    const hidden = () => {
      revision++;
      shellVisible = false;
      output.visible = false;
      following = true;
      void output.open("");
    };
    readVisibility = async () => {
      const request = ++revision;
      if (document.hidden) {
        hidden();
        return;
      }
      try {
        // SettingsVisible is the existing main-window visibility binding;
        // minimization is separately excluded from its reusable-window state.
        const [shown, minimized] = await Promise.all([
          SettingsVisible(),
          Window.IsMinimised(),
        ]);
        if (!alive || request !== revision) return;
        visibilityError = "";
        shellVisible = shown && !minimized && !document.hidden;
      } catch {
        if (!alive || request !== revision) return;
        hidden();
        visibilityError = "Could not open runtime output.";
      }
    };
    const visibilityChanged = () => {
      if (document.hidden) hidden();
      else void readVisibility();
    };
    const subscriptions = [
      Events.On("shell:hidden", hidden),
      Events.On(Events.Types.Common.WindowHide, hidden),
      Events.On(Events.Types.Common.WindowMinimise, hidden),
      ...[
        Events.Types.Common.WindowShow,
        Events.Types.Common.WindowRestore,
        Events.Types.Common.WindowUnMinimise,
        Events.Types.Common.WindowFocus,
      ].map((event) => Events.On(event, () => void readVisibility())),
    ];
    document.addEventListener("visibilitychange", visibilityChanged);
    void readVisibility();
    return () => {
      mounted = false;
      alive = false;
      revision++;
      readVisibility = async () => {};
      for (const off of subscriptions) off();
      document.removeEventListener("visibilitychange", visibilityChanged);
    };
  });

  async function retry() {
    if (!mounted || document.hidden) return;
    const wasVisible = shellVisible;
    await readVisibility();
    if (mounted && wasVisible && shellVisible) void output.open(instanceID);
  }
</script>

<div class="relative flex min-h-0 flex-1 flex-col">
  {#if !instanceID}
    <p
      class="flex h-full items-center justify-center px-3 text-center text-xs text-muted-foreground"
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
    >
      {#if !visibilityError && !output.error && !output.chunks.length}
        <p
          class="pointer-events-none absolute inset-x-3 top-3 text-xs text-muted-foreground"
          role="status"
        >
          {output.busy
            ? "Opening output…"
            : output.accepted
              ? "Waiting for runtime output…"
              : ""}
        </p>
      {/if}
    </ProcessOutputTerminal>
    {#if visibilityError || output.error}
      <div
        class="flex shrink-0 items-center gap-2 border-t border-hairline px-3 py-1.5"
      >
        <p class="min-w-0 flex-1 text-xs text-destructive" role="alert">
          {visibilityError || output.error}
        </p>
        <Button
          variant="ghost"
          size="xs"
          disabled={output.busy}
          onclick={() => void retry()}>Retry output</Button
        >
      </div>
    {/if}
  {/if}
</div>
