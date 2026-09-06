<script lang="ts">
  import { ID, type Profile } from "$bindings/modelprofile";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import { Badge } from "$lib/components/ui/badge";
  import * as Select from "$lib/components/ui/select";

  let {
    id,
    value,
    profiles,
    onChange,
    disabled = false,
  }: {
    id: string;
    value: string;
    profiles: Profile[];
    onChange: (id: ID) => void;
    disabled?: boolean;
  } = $props();
  const selected = $derived(profiles.find((p) => p.id === (value || ID.Generic)));
  function select(value: string) {
    const profile = profiles.find((p) => p.id === value);
    if (profile) onChange(profile.id);
  }
</script>

{#if profiles.length !== 1 || !selected || selected.id !== ID.Generic}
  <ValueRow
    {id}
    label="Model profile"
    hint={selected?.description ?? "This model profile is unavailable for the selected connection."}
  >
    {#snippet control()}
      {#if profiles.length === 1 && selected}
        <div class="flex justify-end"><Badge {id} variant="outline">{selected.name}</Badge></div>
      {:else}
        <Select.Root type="single" value={value || ID.Generic} onValueChange={select} {disabled}>
          <Select.Trigger {id} class="w-full" aria-label="Model profile">
            {selected?.name ?? "Choose model profile"}
          </Select.Trigger>
          <Select.Content>
            {#each profiles as profile (profile.id)}
              <Select.Item value={profile.id} label={profile.name}>{profile.name}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      {/if}
    {/snippet}
  </ValueRow>
{/if}
{#if selected?.language || selected?.reasoningOffRequired}
  <div class="flex flex-wrap items-center gap-2 px-5 pb-4 text-xs text-muted-foreground">
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
