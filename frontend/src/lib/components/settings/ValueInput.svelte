<script lang="ts">
  import { getContext } from "svelte";
  import {
    SETTINGS_VALIDATION,
    type SettingsValidationContext,
  } from "$lib/utils/settingsValidation";
  import type {
    HTMLInputAttributes,
    HTMLInputTypeAttribute,
  } from "svelte/elements";
  import { Input } from "$lib/components/ui/input";
  import { cn } from "$lib/utils";

  type ValueInputProps = Omit<
    HTMLInputAttributes,
    "type" | "value" | "files"
  > & {
    value?: string | number;
    type?: Exclude<HTMLInputTypeAttribute, "file">;
    mono?: boolean;
  };

  let {
    value = $bindable(),
    mono = true,
    type,
    class: className,
    ...rest
  }: ValueInputProps = $props();
  const validation = getContext<SettingsValidationContext | undefined>(
    SETTINGS_VALIDATION,
  );
  const issue = $derived(
    validation?.issue?.control === rest.id ? validation?.issue : null,
  );
</script>

<Input
  bind:value
  {type}
  class={cn(mono && "font-mono text-sm tracking-[-0.01em]", className)}
  {...rest}
  aria-invalid={issue ? true : rest["aria-invalid"]}
  aria-describedby={[
    rest["aria-describedby"],
    issue ? "settings-validation-message" : undefined,
  ]
    .filter(Boolean)
    .join(" ") || undefined}
  oninput={(event) => {
    if (issue) validation?.clear();
    rest.oninput?.(event);
  }}
/>
