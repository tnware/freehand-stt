<script lang="ts">
  import { ID, type Profile } from "$bindings/modelprofile";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import FieldHelp from "./FieldHelp.svelte";
  import { getContext } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import { Badge } from "$lib/components/ui/badge";
  import * as Select from "$lib/components/ui/select";

  let {
    id,
    value,
    profiles,
    onChange,
    disabled = false,
    compact = false,
  }: {
    id: string;
    value: string;
    profiles: Profile[];
    onChange: (id: ID) => void;
    disabled?: boolean;
    compact?: boolean;
  } = $props();
  const validation = getContext<SettingsValidationContext | undefined>(SETTINGS_VALIDATION);
  const issue = $derived(validation?.issue?.control === id ? validation.issue : null);
  const selected = $derived(profiles.find((p) => p.id === (value || ID.Generic)));
  function select(value: string) {
    const profile = profiles.find((p) => p.id === value);
    if (profile) onChange(profile.id);
  }
</script>

{#if profiles.length !== 1 || !selected || selected.id !== ID.Generic}
  <div class={compact ? "space-y-2" : "space-y-2 px-5 py-3"}>
    <div class="flex items-center gap-1">
      <label for={id} class="text-sm font-medium">Model profile</label>
      {#if selected?.description}<FieldHelp
          label="About this model profile"
          text={selected.description}
        />{/if}
    </div>
    {#if profiles.length === 1 && selected}
      <div
        class="flex min-h-9 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm"
        {id}
      >
        <ProviderIcon profile={selected.id} size={18} />{selected.name}
      </div>
    {:else}
      <Select.Root type="single" value={value || ID.Generic} onValueChange={select} {disabled}>
        <Select.Trigger
          {id}
          class="w-full"
          aria-label="Model profile"
          aria-invalid={!!issue}
          aria-describedby={issue ? `${id}-issue` : !selected ? `${id}-unavailable` : undefined}
        >
          <span class="flex min-w-0 items-center gap-2">
            {#if selected && selected.id !== ID.Generic}<ProviderIcon
                profile={selected.id}
                size={18}
              />{/if}
            <span class="truncate">{selected?.name ?? "Choose model profile"}</span>
          </span>
        </Select.Trigger>
        <Select.Content class="w-(--bits-select-anchor-width) max-w-[calc(100vw-24px)]">
          {#each profiles as profile (profile.id)}
            <Select.Item
              value={profile.id}
              label={profile.name}
              class="items-start gap-2.5 py-2.5 pl-3 pr-8 [&>span:last-child]:min-w-0 [&>span:last-child]:shrink [&>span:last-child]:items-start"
            >
              <span class="grid w-[18px] shrink-0 place-items-center pt-0.5">
                {#if profile.id !== ID.Generic}<ProviderIcon profile={profile.id} size={18} />{/if}
              </span>
              <span class="min-w-0">
                <span class="block font-medium">{profile.name}</span>
                <span
                  class="mt-0.5 block text-xs leading-relaxed text-muted-foreground whitespace-normal"
                  >{profile.description}</span
                >
              </span>
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    {/if}
    {#if !selected}<p id={`${id}-unavailable`} class="text-xs text-warning">
        This model profile is unavailable for the selected connection.
      </p>{/if}
    {#if issue}<p id={`${id}-issue`} class="text-xs text-destructive">{issue.message}</p>{/if}
  </div>
{/if}
{#if selected?.language || selected?.reasoningOffRequired}
  <div
    class={compact
      ? "flex flex-wrap items-center gap-2 text-xs text-muted-foreground"
      : "flex flex-wrap items-center gap-2 px-5 pb-3 text-xs text-muted-foreground"}
  >
    {#if selected.language}<Badge variant="outline"
        >{selected.language === "en" ? "English only" : selected.language}</Badge
      >{/if}
    {#if selected.reasoningOffRequired}<Badge variant="outline">Reasoning off</Badge>
      <span
        >{selected.capabilities.cleanupDisableReasoning
          ? "Enforced on each request."
          : "Disable reasoning on the server; this backend has no request override."}</span
      >
    {/if}
  </div>
{/if}
