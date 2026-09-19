<script lang="ts">
  import ButtonIcon from "$lib/components/ui/button/ButtonIcon.svelte";
  import SaveIcon from "@lucide/svelte/icons/save";

  import type { Purpose } from "$bindings/savedconnection";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import { Button } from "$lib/components/ui/button";

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
      : !draft.details.managedInstanceID && !draft.details.baseURL.trim()
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
    <p
      id={explanationID}
      class="mr-auto max-w-full text-xs text-muted-foreground"
    >
      {reason}
    </p>
  {/if}
  <Button type="button" variant="outline" disabled={busy} onclick={onBack}
    >Cancel</Button
  >
  <Button
    type="submit"
    aria-busy={editor.saving}
    form={formID}
    disabled={busy || !!reason}
    aria-describedby={reason ? explanationID : undefined}
  >
    <ButtonIcon icon={SaveIcon} busy={editor.saving} />
    {editor.saving
      ? "Saving…"
      : activateFor
        ? "Save and return"
        : "Save connection"}
  </Button>
</div>
