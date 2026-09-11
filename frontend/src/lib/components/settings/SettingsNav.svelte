<script lang="ts">
  import { GROUP_LABELS, SETTINGS_GROUPS, matchingSettingsSections } from "$lib/navigation";
  import type { SettingsSectionID } from "$lib/navigation";
  import SearchIcon from "@lucide/svelte/icons/search";
  import XIcon from "@lucide/svelte/icons/x";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
  import { cn } from "$lib/utils";

  let {
    active,
    onSelect,
    invalidSection,
    navigationRef = $bindable(null),
  }: {
    active: SettingsSectionID;
    onSelect: (id: SettingsSectionID) => void;
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
        navigationRef?.querySelector<HTMLElement>(`[data-settings-section="${id}"]`)?.focus(),
      );
    } else if (event.key === "ArrowDown" && matches.length) {
      event.preventDefault();
      navigationRef
        ?.querySelector<HTMLElement>(`[data-settings-section="${matches[0].id}"]`)
        ?.focus();
    }
  }

  // Compact navigation keeps section icons; full width adds labels.
  // The selected row uses the same accent wash as workspace navigation.
  const itemClass = (id: SettingsSectionID) =>
    cn(
      "flex min-h-9 w-full items-center justify-center gap-2.5 rounded-lg px-0 text-[13px] transition-colors sm:justify-start sm:px-3",
      "focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
      id === active
        ? "bg-accent-wash font-medium text-accent-text"
        : "text-secondary-foreground hover:bg-subtle-fill-hover hover:text-foreground active:bg-subtle-fill-pressed",
    );

  function moveSelection(event: KeyboardEvent, current: SettingsSectionID) {
    const keys = ["ArrowDown", "ArrowRight", "ArrowUp", "ArrowLeft", "Home", "End"];
    if (!keys.includes(event.key)) return;

    event.preventDefault();
    const currentIndex = matches.findIndex((section) => section.id === current);
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? matches.length - 1
          : (currentIndex +
              (event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : -1) +
              matches.length) %
            matches.length;
    const next = matches[nextIndex];
    if (next.id !== "connections") onSelect(next.id);
    queueMicrotask(() => {
      navigationRef?.querySelector<HTMLElement>(`[data-settings-section="${next.id}"]`)?.focus();
    });
  }
</script>

<nav
  bind:this={navigationRef}
  aria-label="Settings sections"
  class="flex min-h-0 w-14 shrink-0 flex-col gap-5 overflow-y-auto overscroll-contain border-r border-hairline bg-layer-fill/50 px-2 py-5 sm:w-56 sm:px-3"
>
  <p class="hidden px-3 font-display text-lg font-semibold tracking-tight sm:block">Settings</p>
  <p id="settings-nav-help" class="sr-only">
    Use the arrow keys to move between settings sections. Press Home or End to jump to the first or
    last section.
  </p>
  <div class="relative hidden shrink-0 sm:block">
    <SearchIcon
      class="pointer-events-none absolute left-2.5 top-2.5 size-3.5 text-muted-foreground"
    />
    <input
      aria-label="Find settings"
      placeholder="Find settings…"
      bind:value={query}
      onkeydown={searchKey}
      class="h-9 w-full rounded-md border border-input bg-background pl-8 pr-8 text-xs outline-none focus-visible:ring-2 focus-visible:ring-ring"
    />
    {#if query}<TooltipButton
        label="Clear settings search"
        class="absolute right-1 top-1.5"
        onclick={() => (query = "")}><XIcon /></TooltipButton
      >{/if}
  </div>
  {#if !matches.length}<p class="px-2 text-xs text-muted-foreground" role="status">
      No matching settings.
    </p>{/if}
  {#each groups as group (group)}
    {@const sections = matches.filter((section) => section.group === group)}
    {#if sections.length}
      <div class="flex shrink-0 flex-col gap-1">
        <p class="caption hidden px-2.5 pb-1.5 sm:block">{GROUP_LABELS[group]}</p>
        {#each sections as section (section.id)}
          <button
            type="button"
            class={itemClass(section.id)}
            aria-current={section.id === active ? "page" : undefined}
            aria-describedby={section.id === active ? "settings-nav-help" : undefined}
            tabindex={section.id === tabStop ? 0 : -1}
            data-settings-section={section.id}
            aria-label={`${section.label}. ${section.blurb}${invalidSection === section.id ? " Needs attention." : ""}`}
            title={section.label}
            onclick={() => choose(section.id)}
            onkeydown={(event) => moveSelection(event, section.id)}
          >
            <section.icon class="size-[15px] shrink-0" />
            <span class="hidden truncate sm:inline">{section.label}</span>
            {#if section.id === "connections"}<ExternalLinkIcon
                class="ml-auto hidden size-3 text-muted-foreground sm:block"
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
</nav>
