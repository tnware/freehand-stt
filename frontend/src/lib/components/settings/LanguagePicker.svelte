<script lang="ts">
  import { untrack } from "svelte";
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
    immediate = false,
    unavailableReason = "",
  }: {
    id: string;
    value?: string;
    languages: Option[];
    restricted?: boolean;
    disabled?: boolean;
    /** Commit a custom value on blur or Enter instead of on every keystroke. */
    immediate?: boolean;
    unavailableReason?: string;
  } = $props();
  let query = $state("");
  let open = $state(false);
  let custom = $state(false);
  let customDraft = $state<string | null>(null);
  $effect(() => {
    void value;
    untrack(() => (customDraft = null));
  });
  function commitCustom() {
    if (!immediate || disabled || customDraft === null || customDraft === value)
      return;
    value = customDraft;
  }
  const choices = $derived(
    restricted
      ? languages.map((l) => ({
          value: l.code || "__default",
          label: l.label,
          code: l.code,
        }))
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
      (restricted && languages.some((language) => language.code === "auto")
        ? "auto"
        : "__default"),
  );
  const known = $derived(
    choices.find((choice) => choice.value === selectedValue),
  );
  const customVisible = $derived(!restricted && (custom || !known));
  const selected = $derived(customVisible ? "__custom" : selectedValue);
  const selectedLabel = $derived(
    customVisible
      ? "Custom server value…"
      : (known?.label ?? (restricted ? "Choose language" : "Server default")),
  );
  const automaticOnly = $derived(
    restricted && languages.length === 1 && languages[0]?.code === "auto",
  );
  const filtered = $derived(
    choices.filter(
      (choice) =>
        choice.value === "__custom" ||
        `${choice.label} ${choice.code}`
          .toLowerCase()
          .includes(query.toLowerCase().trim()),
    ),
  );

  function choose(next: string) {
    if (disabled) return;
    const choice = choices.find((item) => item.value === next);
    if (!choice) return;
    custom = next === "__custom";
    if (!custom) value = choice.code;
    query = "";
  }
</script>

<div class="flex flex-col gap-2">
  {#if restricted && !known}<p role="status" class="text-xs text-warning">
      The saved language ({value}) is unavailable for this model. Choose a
      supported language before transcribing.
    </p>{/if}
  {#if automaticOnly && known}
    <input
      {id}
      type="text"
      value={selectedLabel}
      readonly
      {disabled}
      aria-describedby={`${id}-help`}
      class="h-8 w-full rounded-md border border-input bg-well px-3 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
    />
  {:else}
    <Combobox.Root
      type="single"
      bind:value={() => selected, choose}
      inputValue={open ? query : selectedLabel}
      bind:open
      items={choices}
      onOpenChange={(isOpen) => {
        if (!isOpen) query = "";
      }}
      allowDeselect={false}
      {disabled}
    >
      <div class="relative">
        <Combobox.Input
          data-slot="combobox-input"
          {id}
          class="h-8 w-full rounded-md border border-input bg-well px-3 pr-9 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
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
          class="picker-trigger"
          aria-label="Show languages"
          ><ChevronsUpDownIcon class="size-4" /></Combobox.Trigger
        >
      </div>
      <Combobox.Portal>
        <Combobox.Content
          data-slot="combobox-content"
          class="picker-content"
          sideOffset={4}
          collisionPadding={12}
        >
          <!-- Recreate filtered options so keyboard highlighting cannot retain a reused DOM node. -->
          {#key query}
            {#each filtered as choice (choice.value)}
              <Combobox.Item
                value={choice.value}
                label={choice.label}
                class="picker-item justify-between"
              >
                <span class="min-w-0 flex-1 break-words">{choice.label}</span>
                {#if selected === choice.value}<CheckIcon
                    class="size-3.5 shrink-0"
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
          <p
            class="mt-1 border-t border-hairline px-3 py-2 text-xs leading-relaxed text-muted-foreground"
          >
            {restricted
              ? "Language selection options for this model profile."
              : "Choose a spoken language, or a custom value from your server. This does not translate audio."}
          </p>
        </Combobox.Content>
      </Combobox.Portal>
    </Combobox.Root>
  {/if}
  {#if customVisible}
    <label class="text-xs text-muted-foreground" for={`${id}-custom`}
      >Custom language value</label
    >
    <ValueInput
      id={`${id}-custom`}
      bind:value={
        () => (immediate ? (customDraft ?? value) : value),
        (next) => {
          if (immediate) customDraft = String(next ?? "");
          else value = String(next ?? "");
        }
      }
      onchange={commitCustom}
      onkeydown={(event) => {
        if (immediate && event.key === "Enter") {
          event.preventDefault();
          commitCustom();
        }
      }}
      maxlength={32}
      {disabled}
      placeholder="Server-specific language code"
      spellcheck={false}
    />
  {/if}
  <p
    id={`${id}-help`}
    class={unavailableReason || customVisible || automaticOnly
      ? "text-xs leading-relaxed text-muted-foreground"
      : "sr-only"}
  >
    {#if unavailableReason}{unavailableReason}
    {:else if customVisible}Custom values are sent to the server unchanged.
    {:else if automaticOnly}This model detects the spoken language
      automatically. Choosing a specific language is not supported.
    {:else}Search by language name or code. {restricted
        ? "Choose from this model profile’s supported languages."
        : "Your model must support the selected language."}{/if}
  </p>
</div>
