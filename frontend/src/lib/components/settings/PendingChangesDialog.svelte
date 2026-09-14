<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";

  let {
    open,
    busy,
    title,
    description,
    error = "",
    errorClass = "text-destructive",
    discardLabel = "Discard",
    discardVariant = "ghost",
    saveLabel = "Save",
    onKeepEditing,
    onDiscard,
    onSave,
  }: {
    open: boolean;
    busy: boolean;
    title: string;
    description: string;
    error?: string;
    errorClass?: string;
    discardLabel?: string;
    discardVariant?: "ghost" | "secondary";
    saveLabel?: string;
    onKeepEditing: () => void;
    onDiscard: () => void;
    onSave: () => void;
  } = $props();

  function preventDismissWhileBusy(event: Event) {
    // onOpenChange observes a close; these hooks can prevent it.
    if (busy) event.preventDefault();
  }
</script>

<Dialog.Root
  {open}
  onOpenChange={(nextOpen) => {
    if (!nextOpen && !busy) onKeepEditing();
  }}
>
  <Dialog.Content
    showCloseButton={!busy}
    onEscapeKeydown={preventDismissWhileBusy}
    onInteractOutside={preventDismissWhileBusy}
  >
    <Dialog.Header>
      <Dialog.Title>{title}</Dialog.Title>
      <Dialog.Description>{description}</Dialog.Description>
    </Dialog.Header>
    {#if error}<p role="alert" class={errorClass}>{error}</p>{/if}
    <Dialog.Footer>
      <Button variant="outline" disabled={busy} onclick={onKeepEditing}
        >Keep editing</Button
      >
      <Button variant={discardVariant} disabled={busy} onclick={onDiscard}
        >{discardLabel}</Button
      >
      <Button disabled={busy} onclick={onSave}>{saveLabel}</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
