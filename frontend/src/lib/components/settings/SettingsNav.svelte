<script lang="ts">
  import {
    GROUP_LABELS,
    SETTINGS_GROUPS,
    SETTINGS_SECTIONS,
    sectionsInGroup,
  } from "$lib/navigation";
  import type { SettingsSectionID } from "$lib/navigation";
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

  // Compact navigation keeps the recognizable section icons. At full width the
  // icon gives way to the label, and the active row is marked by an accent edge
  // rather than a floating card: the nav is chrome, not content.
  const itemClass = (id: SettingsSectionID) =>
    cn(
      "flex min-h-9 w-full items-center justify-center gap-2.5 rounded-md px-0 text-[13px] transition-colors sm:justify-start sm:px-2.5",
      "focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
      id === active
        ? "bg-control-fill font-medium text-foreground shadow-[inset_2px_0_0_var(--primary)]"
        : "text-secondary-foreground hover:bg-subtle-fill-hover hover:text-foreground active:bg-subtle-fill-pressed",
    );

  function moveSelection(event: KeyboardEvent, current: SettingsSectionID) {
    const keys = ["ArrowDown", "ArrowRight", "ArrowUp", "ArrowLeft", "Home", "End"];
    if (!keys.includes(event.key)) return;

    event.preventDefault();
    const currentIndex = SETTINGS_SECTIONS.findIndex((section) => section.id === current);
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? SETTINGS_SECTIONS.length - 1
          : (currentIndex +
              (event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : -1) +
              SETTINGS_SECTIONS.length) %
            SETTINGS_SECTIONS.length;
    const next = SETTINGS_SECTIONS[nextIndex];
    onSelect(next.id);
    queueMicrotask(() => {
      navigationRef?.querySelector<HTMLElement>(`[data-settings-section="${next.id}"]`)?.focus();
    });
  }
</script>

<nav
  bind:this={navigationRef}
  aria-label="Settings sections"
  class="flex min-h-0 w-14 shrink-0 flex-col gap-4 overflow-y-auto overscroll-contain border-r border-hairline bg-layer-fill px-2 py-4 sm:w-56 sm:px-2.5"
>
  <p id="settings-nav-help" class="sr-only">
    Use the arrow keys to move between settings sections. Press Home or End to jump to the first or
    last section.
  </p>
  {#each groups as group (group)}
    <div class="flex shrink-0 flex-col gap-1">
      <p class="caption hidden px-2.5 pb-1.5 sm:block">{GROUP_LABELS[group]}</p>
      {#each sectionsInGroup(group) as section (section.id)}
        <button
          type="button"
          class={itemClass(section.id)}
          aria-current={section.id === active ? "page" : undefined}
          aria-describedby={section.id === active ? "settings-nav-help" : undefined}
          tabindex={section.id === active ? 0 : -1}
          data-settings-section={section.id}
          aria-label={`${section.label}. ${section.blurb}${invalidSection === section.id ? " Needs attention." : ""}`}
          title={section.label}
          onclick={() => onSelect(section.id)}
          onkeydown={(event) => moveSelection(event, section.id)}
        >
          <section.icon class="size-[15px] shrink-0" />
          <span class="hidden truncate sm:inline">{section.label}</span>
          {#if invalidSection === section.id}<span
              class="font-semibold text-destructive"
              aria-hidden="true">!</span
            >{/if}
        </button>
      {/each}
    </div>
  {/each}
</nav>
