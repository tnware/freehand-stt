<script lang="ts">
  import { Combobox } from "bits-ui";
  import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
  import CheckIcon from "@lucide/svelte/icons/check";
  import type { Option } from "$bindings/speechlanguage";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";

  let {
    id,
    value = $bindable(""),
    languages,
    restricted = false,
    disabled = false,
  }: {
    id: string;
    value?: string;
    languages: Option[];
    restricted?: boolean;
    disabled?: boolean;
  } = $props();
  let query = $state("");
  let open = $state(false);
  let custom = $state(false);
  const choices = $derived(
    restricted
      ? languages.map((l) => ({ value: l.code, label: l.label, code: l.code }))
      : [
          { value: "__default", label: "Server default", code: "" },
          { value: "auto", label: "Automatic detection", code: "auto" },
          ...languages.map((language) => ({
            value: language.code,
            label: `${language.label} (${language.code})`,
            code: language.code,
          })),
          { value: "__custom", label: "Custom server value…", code: "" },
        ],
  );
  const selectedValue = $derived(
    value ||
      (restricted && languages.some((language) => language.code === "auto") ? "auto" : "__default"),
  );
  const known = $derived(choices.find((choice) => choice.value === selectedValue));
  const customVisible = $derived(!restricted && (custom || !known));
  const selected = $derived(customVisible ? "__custom" : selectedValue);
  const selectedLabel = $derived(
    customVisible
      ? "Custom server value…"
      : (known?.label ?? (restricted ? "Choose language" : "Server default")),
  );
  const filtered = $derived(
    choices.filter((choice) => choice.label.toLowerCase().includes(query.toLowerCase().trim())),
  );

  function choose(next: string) {
    const choice = choices.find((item) => item.value === next);
    if (!choice) return;
    custom = next === "__custom";
    if (!custom) value = choice.code;
    query = "";
  }
</script>

<div class="flex flex-col gap-2">
  <Combobox.Root
    type="single"
    value={selected}
    inputValue={open ? query : selectedLabel}
    bind:open
    items={choices}
    onValueChange={choose}
    onOpenChange={(isOpen) => {
      if (!isOpen) query = "";
    }}
    allowDeselect={false}
    {disabled}
  >
    <div class="relative">
      <Combobox.Input
        {id}
        class="h-9 w-full rounded-md border border-input bg-background px-3 pr-9 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
        aria-describedby={`${id}-help`}
        placeholder="Search languages…"
        oninput={(event) => {
          query = event.currentTarget.value;
          open = true;
        }}
      >
        {#snippet child({ props })}
          <input {...props} value={open ? query : selectedLabel} />
        {/snippet}
      </Combobox.Input>
      <Combobox.Trigger
        class="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-muted-foreground"
        aria-label="Show languages"><ChevronsUpDownIcon class="size-4" /></Combobox.Trigger
      >
    </div>
    <Combobox.Portal>
      <Combobox.Content
        class="z-50 max-h-[min(18rem,var(--bits-combobox-content-available-height))] w-[var(--bits-combobox-anchor-width)] min-w-56 max-w-[calc(100vw-24px)] overflow-y-auto overscroll-contain rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
        sideOffset={4}
        collisionPadding={12}
      >
        <!-- Recreate filtered options so keyboard highlighting cannot retain a reused DOM node. -->
        {#key query}
          {#each filtered as choice (choice.value)}
            <Combobox.Item
              value={choice.value}
              label={choice.label}
              class="flex cursor-default items-center justify-between gap-2 rounded-sm px-2 py-2 text-sm outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
            >
              {choice.label}{#if selected === choice.value}<CheckIcon
                  class="size-4 shrink-0"
                />{/if}
            </Combobox.Item>
          {:else}
            <p class="px-2 py-3 text-xs text-muted-foreground" role="status">
              {restricted
                ? "No matching language in this model profile."
                : "No matching language. Use Custom server value for an unlisted value."}
            </p>
          {/each}
        {/key}
      </Combobox.Content>
    </Combobox.Portal>
  </Combobox.Root>
  {#if customVisible}
    <label class="text-xs text-muted-foreground" for={`${id}-custom`}>Custom language value</label>
    <ValueInput
      id={`${id}-custom`}
      bind:value
      maxlength={32}
      {disabled}
      placeholder="Server-specific language code"
      spellcheck={false}
    />
  {/if}
  <p id={`${id}-help`} class="text-xs leading-relaxed text-muted-foreground">
    {#if restricted}
      Language choices follow the selected model profile.
    {:else}
      Search by language name or code. Your model must support the language. Custom values are sent
      unchanged.
    {/if}
  </p>
</div>
