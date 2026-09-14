<script lang="ts">
  import type { Component } from "svelte";
  import { tick } from "svelte";
  import * as Dialog from "$lib/components/ui/dialog";
  import SearchIcon from "@lucide/svelte/icons/search";

  export type Command = {
    id: string;
    label: string;
    group: string;
    /** Extra words people actually type for this, beyond the label. */
    keywords?: string;
    /** The current value, shown right-aligned: which model, which connection. */
    detail?: string;
    icon?: Component;
    disabled?: boolean;
    run: () => void;
  };

  let {
    open = $bindable(false),
    commands,
  }: { open?: boolean; commands: Command[] } = $props();

  let query = $state("");
  let index = $state(0);
  let input = $state<HTMLInputElement | null>(null);

  const matches = $derived.by(() => {
    const needle = query.trim().toLowerCase();
    const usable = commands.filter((command) => !command.disabled);
    if (!needle) return usable.slice(0, 12);
    return usable
      .filter((command) =>
        `${command.label} ${command.group} ${command.keywords ?? ""}`
          .toLowerCase()
          .includes(needle),
      )
      .slice(0, 12);
  });

  $effect(() => {
    matches.length;
    index = 0;
  });

  $effect(() => {
    if (!open) {
      query = "";
      index = 0;
    } else void tick().then(() => input?.focus());
  });

  function choose(command: Command | undefined) {
    if (!command) return;
    open = false;
    command.run();
  }

  function key(event: KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      index = matches.length ? (index + 1) % matches.length : 0;
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      index = matches.length ? (index - 1 + matches.length) % matches.length : 0;
    } else if (event.key === "Enter") {
      event.preventDefault();
      choose(matches[index]);
    }
  }
</script>

<!-- Every action the rail and the chain expose, reachable without leaving the
     keyboard. Nothing here is a new capability: it is the same navigation and
     the same settings, addressed by name. -->
<Dialog.Root bind:open>
  <Dialog.Content
    class="top-[12%] w-[560px] max-w-[calc(100%-2rem)] translate-y-0 gap-0 p-0"
    showCloseButton={false}
  >
    <Dialog.Header class="sr-only">
      <Dialog.Title>Commands</Dialog.Title>
      <Dialog.Description>
        Search Freehand's places, settings and actions.
      </Dialog.Description>
    </Dialog.Header>

    <div
      class="flex h-10 items-center gap-2.5 border-b border-hairline px-3.5"
    >
      <SearchIcon class="size-4 shrink-0 text-muted-foreground" />
      <input
        bind:this={input}
        bind:value={query}
        onkeydown={key}
        class="h-full min-w-0 flex-1 bg-transparent text-[13px] outline-none placeholder:text-ink-quiet"
        placeholder="Run a command or search settings"
        aria-label="Command"
        aria-controls="command-results"
        autocomplete="off"
        spellcheck="false"
      />
    </div>

    <div id="command-results" class="max-h-[320px] overflow-y-auto p-1.5" role="listbox">
      {#each matches as command, position (command.id)}
        {#if position === 0 || matches[position - 1].group !== command.group}
          <p
            class="px-2.5 pt-2 pb-1 text-[10px] font-semibold tracking-[0.08em] text-ink-quiet uppercase"
          >
            {command.group}
          </p>
        {/if}
        <button
          type="button"
          role="option"
          aria-selected={position === index}
          class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left transition-colors {position ===
          index
            ? 'bg-accent-wash text-accent-text'
            : 'text-secondary-foreground hover:bg-subtle-fill-hover'}"
          onmouseenter={() => (index = position)}
          onclick={() => choose(command)}
        >
          {#if command.icon}
            <command.icon class="size-4 shrink-0" aria-hidden="true" />
          {:else}
            <span class="size-4 shrink-0"></span>
          {/if}
          <span class="min-w-0 flex-1 truncate text-[13px]">{command.label}</span>
          {#if command.detail}
            <span class="shrink-0 truncate text-[11px] text-ink-quiet"
              >{command.detail}</span
            >
          {/if}
        </button>
      {:else}
        <p class="px-2.5 py-6 text-center text-[13px] text-muted-foreground">
          Nothing matches “{query}”.
        </p>
      {/each}
    </div>
  </Dialog.Content>
</Dialog.Root>
