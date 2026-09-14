<script lang="ts">
  import {
    GROUP_LABELS,
    SETTINGS_GROUPS,
    matchingSettingsSections,
  } from "$lib/navigation";
  import type { SettingsSectionID } from "$lib/navigation";
  import SearchIcon from "@lucide/svelte/icons/search";
  import XIcon from "@lucide/svelte/icons/x";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
  import { cn } from "$lib/utils";

  let {
    active,
    onSelect,
    onOpenChain,
    invalidSection,
    navigationRef = $bindable(null),
  }: {
    active: SettingsSectionID;
    onSelect: (id: SettingsSectionID) => void;
    /** Returns to the workflow chain, where the same controls save at once. */
    onOpenChain?: () => void;
    invalidSection?: SettingsSectionID;
    navigationRef?: HTMLElement | null;
  } = $props();

  const groups = SETTINGS_GROUPS;
  let query = $state("");
  const matches = $derived(matchingSettingsSections(query));
  const tabStop = $derived(
    matches.some((section) => section.id === active) ? active : matches[0]?.id,
  );
  function choose(id: SettingsSectionID) {
    query = "";
    onSelect(id);
  }
  function searchKey(event: KeyboardEvent) {
    if (event.key === "Escape" && query) {
      event.preventDefault();
      event.stopPropagation();
      query = "";
    } else if (event.key === "Enter" && matches.length) {
      event.preventDefault();
      const id = matches[0].id;
      choose(id);
      queueMicrotask(() =>
        navigationRef
          ?.querySelector<HTMLElement>(`[data-settings-section="${id}"]`)
          ?.focus(),
      );
    } else if (event.key === "ArrowDown" && matches.length) {
      event.preventDefault();
      navigationRef
        ?.querySelector<HTMLElement>(
          `[data-settings-section="${matches[0].id}"]`,
        )
        ?.focus();
    }
  }

  // Compact navigation keeps section icons; full width adds labels.
  // The selected row uses the same accent wash as workspace navigation.
  const itemClass = (id: SettingsSectionID) =>
    cn(
      "flex min-h-9 w-full items-center justify-center gap-2.5 rounded-lg px-0 text-[13px] transition-colors min-[760px]:justify-start min-[760px]:px-3",
      "focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
      id === active
        ? "bg-accent-wash font-semibold text-accent-text ring-1 ring-inset ring-primary/20"
        : "text-secondary-foreground hover:bg-subtle-fill-hover hover:text-foreground active:bg-subtle-fill-pressed",
    );

  function moveSelection(event: KeyboardEvent, current: SettingsSectionID) {
    const keys = [
      "ArrowDown",
      "ArrowRight",
      "ArrowUp",
      "ArrowLeft",
      "Home",
      "End",
    ];
    if (!keys.includes(event.key)) return;

    event.preventDefault();
    const currentIndex = matches.findIndex((section) => section.id === current);
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? matches.length - 1
          : (currentIndex +
              (event.key === "ArrowDown" || event.key === "ArrowRight"
                ? 1
                : -1) +
              matches.length) %
            matches.length;
    const next = matches[nextIndex];
    if (next.id !== "connections") onSelect(next.id);
    queueMicrotask(() => {
      navigationRef
        ?.querySelector<HTMLElement>(`[data-settings-section="${next.id}"]`)
        ?.focus();
    });
  }
</script>

<nav
  bind:this={navigationRef}
  aria-label="Settings sections"
  class="flex min-h-0 w-14 shrink-0 flex-col gap-3 overflow-y-auto overscroll-contain border-r border-hairline bg-layer-fill px-2 pt-0 pb-4 min-[760px]:w-[252px] min-[760px]:px-0"
>
  <p
    class="hidden h-[33px] shrink-0 items-center border-b border-hairline pl-3.5 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase min-[760px]:flex"
  >
    Settings
  </p>
  <p id="settings-nav-help" class="sr-only">
    Use the arrow keys to move between settings sections. Press Home or End to
    jump to the first or last section.
  </p>
  <div class="relative hidden shrink-0 px-1.5 pt-2 min-[760px]:block">
    <SearchIcon
      class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
    />
    <input
      aria-label="Find settings"
      placeholder="Find settings…"
      bind:value={query}
      onkeydown={searchKey}
      class="h-9 w-full rounded-lg border border-input bg-background pl-9 pr-8 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring"
    />
    {#if query}<TooltipButton
        label="Clear settings search"
        class="absolute right-1 top-1/2 -translate-y-1/2"
        onclick={() => (query = "")}><XIcon /></TooltipButton
      >{/if}
  </div>
  {#if !matches.length}<p
      class="px-3.5 text-xs text-muted-foreground"
      role="status"
    >
      No matching settings.
    </p>{/if}
  {#each groups as group (group)}
    {@const sections = matches.filter((section) => section.group === group)}
    {#if sections.length}
      <div class="flex shrink-0 flex-col gap-0.5 px-1.5">
        <p
          class="hidden px-2.5 pb-1 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase min-[760px]:block"
        >
          {GROUP_LABELS[group]}
        </p>
        {#each sections as section (section.id)}
          <button
            type="button"
            class={itemClass(section.id)}
            aria-current={section.id === active ? "page" : undefined}
            aria-describedby={section.id === active
              ? "settings-nav-help"
              : undefined}
            tabindex={section.id === tabStop ? 0 : -1}
            data-settings-section={section.id}
            aria-label={`${section.label}. ${section.blurb}${invalidSection === section.id ? " Needs attention." : ""}`}
            title={section.label}
            onclick={() => choose(section.id)}
            onkeydown={(event) => moveSelection(event, section.id)}
          >
            <section.icon class="size-4 shrink-0" />
            <span class="hidden truncate min-[760px]:inline"
              >{section.label}</span
            >
            {#if section.id === "connections"}<ExternalLinkIcon
                class="ml-auto hidden size-3 text-muted-foreground min-[760px]:block"
              />{/if}
            {#if invalidSection === section.id}<span
                class="font-semibold text-destructive"
                aria-hidden="true">!</span
              >{/if}
          </button>
        {/each}
      </div>
    {/if}
  {/each}

  {#if onOpenChain}
    <div class="mt-auto hidden px-1.5 pt-4 min-[760px]:block">
      <!-- These pages are the full form. The chain carries the same few
           controls inline and saves them immediately, which is where most
           visits actually want to go. -->
      <div
        class="rounded-lg border border-accent-edge bg-accent-wash p-2.5 text-[11.5px] leading-relaxed"
      >
        <p class="text-secondary-foreground">
          Connection, model and cleanup are also on each workflow&#39;s chain,
          where changes save as you make them.
        </p>
        <button
          type="button"
          class="mt-1.5 text-accent-text underline-offset-2 hover:underline"
          onclick={onOpenChain}>Open the Voice chain →</button
        >
      </div>
    </div>
  {/if}
</nav>
