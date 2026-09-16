<script lang="ts">
  import * as Picker from "$lib/components/ui/combobox";
  import type { Snippet } from "svelte";
  import { Combobox } from "bits-ui";
  import { VoiceScope, type VoicesResult } from "$bindings/inference";
  import PickerRefreshButton from "./PickerRefreshButton.svelte";
  import CheckIcon from "@lucide/svelte/icons/check";

  let {
    id,
    value = $bindable(),
    supported = false,
    allowedVoices = [],
    result = null,
    busy = false,
    onDiscover,
    compact = false,
    sidebar = false,
    disabled = false,
    onChoose,
    actions,
  }: {
    actions?: Snippet;
    id: string;
    value: string;
    supported?: boolean;
    allowedVoices?: string[];
    result?: VoicesResult | null;
    busy?: boolean;
    onDiscover: () => void;
    compact?: boolean;
    sidebar?: boolean;
    disabled?: boolean;
    onChoose?: (voice: string) => boolean | Promise<boolean>;
  } = $props();
  let open = $state(false),
    query = $state("");
  let edited = $state(false);
  const voices = $derived(
    allowedVoices.length
      ? allowedVoices.map(
          (id) =>
            result?.voices?.find((v) => v.id === id) ?? {
              id,
              name: id,
              language: "",
            },
        )
      : result?.errorKind
        ? []
        : (result?.voices ?? []),
  );
  const matches = $derived(
    voices.filter((v) =>
      `${v.id} ${v.name} ${v.language}`
        .toLowerCase()
        .includes(query.trim().toLowerCase()),
    ),
  );
  const custom = $derived(
    !allowedVoices.length &&
      query.trim() &&
      !voices.some((v) => v.id === query.trim())
      ? query.trim()
      : "",
  );
  const choices = $derived([
    ...matches.map((v) => ({
      value: v.id,
      label: v.name || v.id,
      language: v.language,
    })),
    ...(custom ? [{ value: custom, label: custom, language: "" }] : []),
  ]);
  const errors: Record<string, string> = {
    unsupported:
      "This backend has no qualified voice discovery. Enter a voice ID manually.",
    credential_missing: "Add an API key in Connections, then refresh voices.",
    credential_unavailable:
      "Re-enter the saved connection’s API key, then refresh voices.",
    invalid_settings: "Check the selected connection and model settings.",
    response:
      "The server returned an unexpected voice list. You can still enter a voice ID.",
    response_too_large:
      "The server’s metadata exceeded the size limit. Enter a voice ID manually.",
    timeout: "Voice discovery timed out. Check the server and try again.",
    tls: "Check the server’s TLS certificate and hostname.",
    dns: "The server name could not be resolved. Check your connection.",
  };
  const failure = $derived(
    !result?.errorKind
      ? ""
      : result.errorKind === "http"
        ? result.httpStatus === 401 || result.httpStatus === 403
          ? "The voice endpoint denied access. Check the connection’s key and permissions."
          : result.httpStatus === 404 || result.httpStatus === 405
            ? "This server version does not expose the voice metadata route. Enter a voice ID manually."
            : `Voice discovery failed (HTTP ${result.httpStatus}). You can still enter a voice ID.`
        : errors[result.errorKind] ||
          "Voice discovery failed. Check your connection and try again.",
  );
  async function choose(next: string) {
    edited = false;
    if (allowedVoices.length && !allowedVoices.includes(next)) return;
    if (next && (!onChoose || (await onChoose(next)))) {
      if (!onChoose) value = next;
      edited = false;
      open = false;
      query = "";
    }
  }
</script>

<div class={compact ? "space-y-2" : "space-y-2 px-5 py-4"}>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <label for={id} class="content-value">Voice</label>
    <div class="flex flex-wrap items-center gap-2">
      {#if supported}<PickerRefreshButton
          label="Refresh voices"
          {sidebar}
          {busy}
          {disabled}
          onclick={onDiscover}
        />{/if}
      {@render actions?.()}
    </div>
  </div>
  <Combobox.Root
    {disabled}
    type="single"
    bind:value={() => value, (next) => void choose(next)}
    inputValue={open ? query : value}
    bind:open
    items={choices}
    onOpenChange={(next) => {
      if (!next) {
        if (edited && !allowedVoices.length) {
          if (onChoose) void onChoose(query.trim());
          else value = query.trim();
        }
        edited = false;
        query = "";
      }
    }}
    allowDeselect={false}
  >
    <div class="relative">
      <Combobox.Input
        data-slot="combobox-input"
        {id}
        aria-label="Choose voice"
        aria-describedby={`${id}-help`}
        placeholder={allowedVoices.length
          ? "Search preset voices…"
          : supported
            ? "Search or enter a voice ID…"
            : "Enter a voice ID…"}
        class="h-8 w-full min-w-0 rounded-md border border-input bg-well px-3 pr-9 font-mono text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
        spellcheck={false}
        maxlength={200}
        oninput={(e) => {
          query = e.currentTarget.value;
          edited = true;
          open = true;
        }}
        onkeydown={(e) => {
          if (e.key === "Escape") {
            edited = false;
            query = "";
          }
          if (
            e.key === "Enter" &&
            query.trim() &&
            !e.currentTarget.getAttribute("aria-activedescendant")
          ) {
            e.preventDefault();
            choose(query.trim());
          }
        }}
      >
        {#snippet child({ props })}<input
            {...props}
            value={open ? query : value}
          />{/snippet}
      </Combobox.Input>
      <Picker.Trigger aria-label="Show voices" />
    </div>
    <Picker.Content>
      {#key query}{#each choices as choice (choice.value)}
          <Picker.Item value={choice.value} label={choice.label}>
            <span class="min-w-0 flex-1 break-all"
              ><span class="font-mono"
                >{custom === choice.value
                  ? `Use “${choice.value}”`
                  : choice.label}</span
              >{#if choice.label !== choice.value}<span
                  class="mt-0.5 block text-xs text-muted-foreground"
                  >{choice.value}</span
                >{/if}</span
            >
            {#if choice.language}<span class="text-xs text-muted-foreground"
                >{choice.language}</span
              >{/if}
            {#if value === choice.value}<CheckIcon
                class="size-3.5 shrink-0"
              />{/if}
          </Picker.Item>
        {:else}<p class="px-3 py-3 text-xs text-muted-foreground">
            {allowedVoices.length
              ? "No matching preset voices."
              : supported
                ? "Refresh voices, or enter a voice ID supplied by your server."
                : "Enter a voice ID supplied by your server."}
          </p>{/each}{/key}
    </Picker.Content>
  </Combobox.Root>
  <p
    id={`${id}-help`}
    role="status"
    class={compact && !result && !busy
      ? "sr-only"
      : `text-xs leading-relaxed ${failure && !busy ? "text-warning" : "text-muted-foreground"}`}
  >
    {#if busy}Loading voices…
    {:else if allowedVoices.length}{voices.length} preset voices for this model profile.
      {failure
        ? "Server voice refresh failed; the preset list remains available."
        : ""}
    {:else if failure}{failure}
    {:else if result}{voices.length}
      {result.scope === VoiceScope.VoiceScopeModel
        ? "voices advertised for this model."
        : "server voices; availability depends on the selected model."}
      {result.truncated ? "List limited to 500 entries. " : ""}Custom voice IDs
      remain available.
    {:else}{supported
        ? "Choose a server voice or enter an ID."
        : "Enter a voice ID supported by your model."}{/if}
  </p>
</div>
