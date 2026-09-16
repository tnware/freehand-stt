<script lang="ts">
  import { getContext, type Snippet } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import FieldCaption from "./FieldCaption.svelte";

  let {
    id,
    label,
    hint = "",
    control,
    action,
  }: {
    id: string;
    label: string;
    hint?: string;
    /** The editable value, using the shared field geometry. */
    control: Snippet;
    /** Optional status or action that directly affects this value. */
    action?: Snippet;
  } = $props();

  const uid = $props.id();
  const labelID = `${uid}-label`;
  const hintID = `${uid}-hint`;
  const errorID = `${uid}-error`;
  const validation = getContext<SettingsValidationContext | undefined>(
    SETTINGS_VALIDATION,
  );
  const issue = $derived(
    validation?.issue?.control === id ? validation?.issue : null,
  );
</script>

<div
  class="grid gap-2 py-3 @min-[600px]:grid-cols-[minmax(0,1fr)_minmax(180px,1fr)] @min-[600px]:items-start @min-[600px]:gap-4"
  role="group"
  aria-labelledby={labelID}
  aria-describedby={[hint ? hintID : null, issue ? errorID : null]
    .filter(Boolean)
    .join(" ") || undefined}
>
  <FieldCaption
    {label}
    controlID={id}
    {labelID}
    description={hint}
    descriptionID={hintID}
    error={issue?.message}
    {errorID}
  />
  <div class="flex min-w-0 items-center justify-end gap-2">
    <div class="min-w-0 flex-1">{@render control()}</div>
    {#if action}<div class="flex shrink-0 items-center gap-2">
        {@render action()}
      </div>{/if}
  </div>
</div>
