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
    unavailableReason = "",
  }: {
    id: string;
    value?: string;
    languages: Option[];
    restricted?: boolean;
    disabled?: boolean;
    unavailableReason?: string;
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
    choices.filter(
      (choice) =>
        choice.value === "__custom" ||
        `${choice.label} ${choice.code}`.toLowerCase().includes(query.toLowerCase().trim()),
    ),
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
        class="h-9 w-full rounded-lg border border-input bg-background px-3 pr-9 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
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
        data-slot="combobox-content"
        class="z-50 max-h-[min(18rem,var(--bits-combobox-content-available-height))] w-[var(--bits-combobox-anchor-width)] min-w-56 max-w-[calc(100vw-24px)] overflow-y-auto overscroll-contain rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-md"
        sideOffset={4}
        collisionPadding={12}
      >
        <!-- Recreate filtered options so keyboard highlighting cannot retain a reused DOM node. -->
        {#key query}
          {#each filtered as choice (choice.value)}
            <Combobox.Item
              value={choice.value}
              label={choice.label}
              class="flex cursor-default items-center justify-between gap-3 rounded-sm px-3 py-2.5 text-sm outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
            >
              <span class="min-w-0 flex-1 break-words">{choice.label}</span>
              {#if selected === choice.value}<CheckIcon class="size-3.5 shrink-0" />{/if}
            </Combobox.Item>
          {:else}
            <p class="px-2 py-3 text-xs text-muted-foreground" role="status">
              {restricted
                ? "No matching language in this model profile."
                : "No matching language. Use Custom server value for an unlisted value."}
            </p>
          {/each}
        {/key}
        <p
          class="mt-1 border-t border-hairline px-3 py-2 text-xs leading-relaxed text-muted-foreground"
        >
          {restricted
            ? "Languages supported by this model profile."
            : "Choose a spoken language, or a custom value from your server. This does not translate audio."}
        </p>
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
  <p
    id={`${id}-help`}
    class={unavailableReason || customVisible
      ? "text-xs leading-relaxed text-muted-foreground"
      : "sr-only"}
  >
    {#if unavailableReason}{unavailableReason}
    {:else if customVisible}Custom values are sent to the server unchanged.
    {:else}Search by language name or code. {restricted
        ? "Choose from this model profile’s supported languages."
        : "Your model must support the selected language."}{/if}
  </p>
</div>
