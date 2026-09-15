<script lang="ts">
  import { getContext, type Snippet } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import { Label } from "$lib/components/ui/label";

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
  aria-describedby={hint ? hintID : undefined}
>
  <div class="min-w-0">
    <Label id={labelID} for={id} class="content-value">{label}</Label>
    {#if hint}
      <p id={hintID} class="content-meta mt-1 max-w-2xl">
        {hint}
      </p>
    {/if}
    {#if issue}<p class="mt-1 text-xs text-destructive">{issue.message}</p>{/if}
  </div>
  <div class="flex min-w-0 items-center justify-end gap-2">
    <div class="min-w-0 flex-1">{@render control()}</div>
    {#if action}<div class="flex shrink-0 items-center gap-2">
        {@render action()}
      </div>{/if}
  </div>
</div>
