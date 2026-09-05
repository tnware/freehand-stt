<script lang="ts">
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
  const selected = $derived(profiles.find((profile) => profile.id === (value || ID.Generic)));
  const available = $derived(profiles.filter((profile) => profile.available));
  const planned = $derived(profiles.filter((profile) => !profile.available));
  function choose(next: string) {
    const profile = available.find((profile) => profile.id === next);
    if (profile) value = profile.id;
  }
</script>

<ValueRow
  {id}
  label="Compatibility profile"
  hint={selected?.description ?? "Choose the server contract for this connection."}
>
  {#snippet control()}
    <Select.Root type="single" value={value || ID.Generic} onValueChange={choose}>
      <Select.Trigger {id} class="w-full">{selected?.label ?? "Choose a profile"}</Select.Trigger>
      <Select.Content class="max-h-80">
        <Select.Group>
          <Select.Label>Available profiles</Select.Label>
          {#each available as profile (profile.id)}
            <Select.Item value={profile.id} label={profile.label}>{profile.label}</Select.Item>
          {/each}
        </Select.Group>
        <Select.Group>
          <Select.Label>Planned profiles — dedicated support not implemented</Select.Label>
          {#each planned as profile (profile.id)}
            <Select.Item value={profile.id} label={profile.label} disabled>
              <div class="max-w-72 whitespace-normal">
                <div>{profile.label}</div>
                <div class="text-xs">{profile.description}</div>
              </div>
            </Select.Item>
          {/each}
        </Select.Group>
      </Select.Content>
    </Select.Root>
  {/snippet}
</ValueRow>
<p class="px-5 pb-3 text-xs leading-relaxed text-muted-foreground">
  A server with a planned profile may already work through Generic. Connection tests read metadata;
  they do not verify model capabilities.
</p>
