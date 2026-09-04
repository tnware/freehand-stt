<script lang="ts">
  import SlidersHorizontalIcon from "@lucide/svelte/icons/sliders-horizontal";
  import WandSparklesIcon from "@lucide/svelte/icons/wand-sparkles";
  import { Badge } from "$lib/components/ui/badge";
  import * as Field from "$lib/components/ui/field";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { PostProcessingPreset, type ProfileDescriptor } from "$lib/state";

  let {
    profiles,
    value = $bindable(),
    disabled = false,
  }: {
    profiles: ProfileDescriptor[];
    value: PostProcessingPreset;
    disabled?: boolean;
  } = $props();

  function selectProfile(id: string) {
    const selected = profiles.find((profile) => profile.id === id);
    if (selected) value = selected.id;
  }
</script>

<Field.Set {disabled}>
  <Field.Legend>Processing behavior</Field.Legend>
  <Field.Description>
    Choose how Freehand builds the cleanup request. The endpoint and model stay independent from
    this behavior.
  </Field.Description>
  <RadioGroup.Root
    {value}
    onValueChange={selectProfile}
    class="grid gap-2.5 sm:grid-cols-2"
    aria-label="Post-processing behavior"
    {disabled}
  >
    {#each profiles as profile (profile.id)}
      <Field.Label for={`processing-profile-${profile.id}`}>
        <Field.Field orientation="horizontal">
          <div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted" aria-hidden="true">
            {#if profile.id === PostProcessingPreset.PostProcessingPresetS1Mini}
              <SlidersHorizontalIcon class="size-4" />
            {:else}
              <WandSparklesIcon class="size-4" />
            {/if}
          </div>
          <Field.Content>
            <div class="flex flex-wrap items-center gap-2">
              <Field.Title>{profile.name}</Field.Title>
              <Badge variant="secondary">
                {profile.instructionEditable ? "Flexible" : "Purpose-built"}
              </Badge>
            </div>
            <Field.Description>{profile.description}</Field.Description>
          </Field.Content>
          <RadioGroup.Item
            id={`processing-profile-${profile.id}`}
            value={profile.id}
          />
        </Field.Field>
      </Field.Label>
    {/each}
  </RadioGroup.Root>
</Field.Set>
