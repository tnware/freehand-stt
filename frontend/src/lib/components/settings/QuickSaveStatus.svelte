<script lang="ts">
  import SaveIndicator from "./SaveIndicator.svelte";
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
  class="flex min-h-4 items-start gap-1.5 text-xs leading-4"
  class:text-destructive={failure && !saving}
  class:text-success={!saving && !failure && !!saved && fields.includes(saved)}
  class:text-muted-foreground={saving ||
    (!failure && (!saved || !fields.includes(saved)))}
  role="status"
  aria-atomic="true"
>
  <span class="inline-flex h-4 shrink-0 items-center">
    <SaveIndicator
      pending={saving}
      failed={failure}
      saved={!!saved && fields.includes(saved)}
    />
  </span>
  <span class="min-w-0 [overflow-wrap:anywhere]">{message}</span>
</p>
