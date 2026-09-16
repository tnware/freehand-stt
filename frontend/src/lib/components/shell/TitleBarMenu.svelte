<script lang="ts">
  import { tick } from "svelte";
  import { Menubar } from "bits-ui";
  import CheckIcon from "@lucide/svelte/icons/check";

  export type TitleMenuItem = {
    id: string;
    label: string;
    disabled?: boolean;
    checked?: boolean;
    shortcut?: string;
    run: () => void;
  };
  export type TitleMenu = {
    id: string;
    label: string;
    groups: TitleMenuItem[][];
  };
  let { menus }: { menus: TitleMenu[] } = $props();
  let active = $state("");

  async function choose(item: TitleMenuItem) {
    if (item.disabled) return;
    active = "";
    // Release the menu and its focus scope before opening a workflow or dialog.
    await tick();
    item.run();
  }
</script>

<Menubar.Root
  bind:value={active}
  aria-label="Application menu"
  class="flex shrink-0 items-center"
  style="--wails-draggable: no-drag"
>
  {#each menus as menu (menu.id)}
    <Menubar.Menu value={menu.id}>
      <Menubar.Trigger
        class="rounded-sm px-2 py-1 text-xs text-muted-foreground outline-none hover:bg-subtle-fill-hover hover:text-foreground data-[highlighted]:bg-subtle-fill-hover data-[state=open]:bg-accent-wash data-[state=open]:text-foreground focus-visible:ring-1 focus-visible:ring-ring"
        >{menu.label}</Menubar.Trigger
      >
      <Menubar.Portal>
        <Menubar.Content
          align="start"
          sideOffset={4}
          class="z-50 max-h-[var(--bits-menubar-content-available-height)] min-w-60 overflow-y-auto rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-float outline-none"
          style="--wails-draggable: no-drag"
        >
          {#each menu.groups as group, index (index)}
            {#if index > 0}<Menubar.Separator
                class="my-1 h-px bg-border"
              />{/if}
            {#each group as item (item.id)}
              {#snippet label()}
                <span class="grid size-4 shrink-0 place-items-center">
                  {#if item.checked}<CheckIcon class="size-3.5" />{/if}
                </span>
                <span class="flex-1">{item.label}</span>
                {#if item.shortcut}<span
                    class="ml-4 text-xs text-muted-foreground"
                    >{item.shortcut}</span
                  >{/if}
              {/snippet}
              {#if item.checked !== undefined}
                <Menubar.CheckboxItem
                  checked={item.checked}
                  disabled={item.disabled}
                  onSelect={() => void choose(item)}
                  class="menu-row"
                  >{@render label()}</Menubar.CheckboxItem
                >
              {:else}
                <Menubar.Item
                  disabled={item.disabled}
                  onSelect={() => void choose(item)}
                  class="menu-row"
                  >{@render label()}</Menubar.Item
                >
              {/if}
            {/each}
          {/each}
        </Menubar.Content>
      </Menubar.Portal>
    </Menubar.Menu>
  {/each}
</Menubar.Root>
