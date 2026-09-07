<script lang="ts">
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import * as Select from "$lib/components/ui/select";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import { ID, type Profile } from "$bindings/compatibility";

  let {
    id,
    value = $bindable(),
    profiles,
  }: {
    id: string;
    value: ID;
    profiles: Profile[];
  } = $props();
  const selected = $derived(
    profiles.find((profile) => profile.id === (value || ID.Generic)),
  );
  const available = $derived(profiles.filter((profile) => profile.available));
  function choose(next: string) {
    const profile = available.find((profile) => profile.id === next);
    if (profile) value = profile.id;
  }
</script>

<ValueRow
  {id}
  label="Compatibility profile"
  hint={selected?.description ??
    "Choose the server contract for this connection."}
>
  {#snippet control()}
    <Select.Root
      type="single"
      value={value || ID.Generic}
      onValueChange={choose}
    >
      <Select.Trigger {id} class="w-full"
        ><span class="flex min-w-0 items-center gap-2"
          ><ProviderIcon profile={selected?.id} size={22} /><span
            class="truncate">{selected?.label ?? "Choose a profile"}</span
          ></span
        ></Select.Trigger
      >
      <Select.Content class="max-h-80">
        <Select.Group>
          <Select.Label>Available profiles</Select.Label>
          {#each available as profile (profile.id)}
            <Select.Item value={profile.id} label={profile.label}
              ><ProviderIcon
                profile={profile.id}
                size={22}
              />{profile.label}</Select.Item
            >
          {/each}
        </Select.Group>
      </Select.Content>
    </Select.Root>
  {/snippet}
</ValueRow>
{#if selected && !selected.available}
  <p class="px-5 pb-3 text-xs leading-relaxed text-muted-foreground">
    This saved profile is unavailable; your selection has not changed. Choose an
    available profile to use this connection. Generic may work if your server
    supports its API.
  </p>
{/if}
<p class="px-5 pb-3 text-xs leading-relaxed text-muted-foreground">
  Connection checks read metadata only; they do not verify model capabilities.
</p>
