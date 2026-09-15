<script lang="ts">
  import type { QuickSettingsField } from "$lib/stores/editor.svelte";
  let {
    fields,
    pending = [],
    saved = null,
    failed = null,
    quiet = false,
  }: {
    fields: QuickSettingsField[];
    pending?: QuickSettingsField[];
    saved?: QuickSettingsField | null;
    failed?: QuickSettingsField | null;
    quiet?: boolean;
  } = $props();
  const saving = $derived(pending.some((field) => fields.includes(field)));
  const failure = $derived(!!failed && fields.includes(failed));
  const message = $derived(
    saving
      ? "Saving…"
      : failure
        ? "Could not save. Your previous settings are still active. Try the change again."
        : saved && fields.includes(saved)
          ? "Saved"
          : quiet
            ? ""
            : "Changes apply immediately to the next request.",
  );
</script>

<p
  class={quiet && !message ? "sr-only" : "min-h-4 text-xs leading-relaxed"}
  class:text-destructive={failure && !saving}
  class:text-muted-foreground={!failure || saving}
  role="status"
>
  {message}
</p>
