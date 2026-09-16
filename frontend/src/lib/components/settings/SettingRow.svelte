<script lang="ts">
  import { getContext, type Snippet } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import FieldCaption from "./FieldCaption.svelte";

  let {
    title,
    compact = false,
    description = "",
    controlID,
    control,
    children,
  }: {
    title: string;
    compact?: boolean;
    description?: string;
    controlID?: string;
    /** Trailing control: a switch, a button, a segmented group. */
    control?: Snippet;
    /** Full-width content below the title, for dependent fields. */
    children?: Snippet;
  } = $props();

  const uid = $props.id();
  const titleID = `${uid}-title`;
  const descriptionID = `${uid}-description`;
  const errorID = `${uid}-error`;
  const validation = getContext<SettingsValidationContext | undefined>(
    SETTINGS_VALIDATION,
  );
  const issue = $derived(
    controlID && validation?.issue?.control === controlID
      ? validation.issue
      : null,
  );
</script>

<div
  class={compact ? "py-2" : "py-3"}
  role="group"
  aria-labelledby={titleID}
  aria-describedby={[description ? descriptionID : null, issue ? errorID : null]
    .filter(Boolean)
    .join(" ") || undefined}
>
  <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
    <div class="min-w-0 flex-[1_1_12rem]">
      <FieldCaption
        label={title}
        {controlID}
        labelID={titleID}
        {description}
        {descriptionID}
        error={issue?.message}
        {errorID}
      />
    </div>
    {#if control}
      <div
        class="ml-auto flex max-w-full shrink-0 flex-wrap items-center gap-2"
      >
        {@render control()}
      </div>
    {/if}
  </div>
  {#if children}
    <div class="mt-2">{@render children()}</div>
  {/if}
</div>
