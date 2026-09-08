<script lang="ts">
  import type { Purpose } from "$bindings/savedconnection";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import { Button } from "$lib/components/ui/button";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let {
    editor,
    activateFor,
    formID,
    onBack,
  }: {
    editor: SettingsEditor;
    activateFor?: Purpose;
    formID?: string;
    onBack: () => void;
  } = $props();

  const draft = $derived(editor.connectionDraft);
  const busy = $derived(editor.saving || editor.managedConnectionTesting);
  const reason = $derived(
    !draft?.name.trim()
      ? "Enter a connection name."
      : !draft.details.baseURL.trim()
        ? "Enter the server’s base URL."
        : draft.uses.length === 0
          ? "Choose at least one use for this connection."
          : activateFor !== undefined && !draft.uses.includes(activateFor)
            ? "Enable this task for the connection."
            : "",
  );
  const explanationID = $props.id();
</script>

<div class="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2">
  {#if reason}
    <p id={explanationID} class="mr-auto max-w-full text-xs text-muted-foreground">
      {reason}
    </p>
  {/if}
  <Button type="button" variant="outline" disabled={busy} onclick={onBack}>Cancel</Button>
  <Button
    type="submit"
    form={formID}
    disabled={busy || !!reason}
    aria-describedby={reason ? explanationID : undefined}
  >
    {#if editor.saving}<LoaderCircleIcon class="animate-spin" />{/if}
    {editor.saving ? "Saving…" : activateFor ? "Save and set up" : "Save connection"}
  </Button>
</div>
