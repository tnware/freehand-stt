<script lang="ts">
  import { onDestroy } from "svelte";
  import { Purpose } from "$bindings/savedconnection";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import ConnectionsSection from "./sections/ConnectionsSection.svelte";
  import * as Dialog from "$lib/components/ui/dialog";
  import { Button } from "$lib/components/ui/button";

  let {
    editor,
    blocked = false,
    purpose,
    error,
    onClose,
  }: {
    editor: SettingsEditor;
    blocked?: boolean;
    purpose: Purpose;
    error: string;
    onClose: () => void;
  } = $props();
  const returnFocus = typeof document === "undefined" ? null : document.activeElement;
  const label = $derived(
    purpose === Purpose.Transcription
      ? "transcription"
      : purpose === Purpose.Cleanup
        ? "cleanup"
        : "text to speech",
  );
  const busy = $derived(editor.saving || editor.managedConnectionTesting);
  let discardOpen = $state(false);
  function close() {
    editor.cancelConnectionEdit();
    onClose();
  }
  function requestClose() {
    if (busy) return;
    if (editor.connectionDirty) discardOpen = true;
    else close();
  }
  // Window-hide handling clears the transient editor draft. Close this entrance too.
  $effect(() => {
    if (!editor.connectionDraft && !editor.saving) onClose();
  });
  onDestroy(() => editor.cancelConnectionEdit());
</script>

<Dialog.Root
  open
  onOpenChange={(open) => {
    if (!open) requestClose();
  }}
>
  <Dialog.Content
    class="flex max-h-[90dvh] flex-col overflow-hidden sm:max-w-[660px]"
    showCloseButton={false}
    onCloseAutoFocus={(event) => {
      event.preventDefault();
      if (returnFocus instanceof HTMLElement && returnFocus.isConnected) returnFocus.focus();
    }}
  >
    <Dialog.Header class="shrink-0">
      <Dialog.Title>Add a connection for {label}</Dialog.Title>
      <Dialog.Description
        >Save and use selects this server for {label}. You’ll return to your task to choose a model.</Dialog.Description
      >
    </Dialog.Header>
    {#if blocked}
      <p role="status">Close the Settings window to continue adding this connection here.</p>
      <Button variant="outline" onclick={requestClose}>Cancel setup</Button>
    {:else}
      <ConnectionsSection
        {editor}
        {error}
        activateFor={purpose}
        onBack={requestClose}
        onSaved={onClose}
        onOpenFeature={() => {}}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>
<Dialog.Root bind:open={discardOpen}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Discard connection changes?</Dialog.Title>
      <Dialog.Description
        >This connection has not been saved. Your current connection will stay selected.</Dialog.Description
      >
    </Dialog.Header>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (discardOpen = false)}>Keep editing</Button>
      <Button variant="destructive" onclick={close}>Discard changes</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
