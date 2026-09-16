<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import ActionDialog from "$lib/components/common/ActionDialog.svelte";
  import SaveIcon from "@lucide/svelte/icons/save";
  import ButtonIcon from "$lib/components/ui/button/ButtonIcon.svelte";

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
</script>

<ActionDialog
  {open}
  {busy}
  {title}
  {description}
  {error}
  {errorClass}
  icon={SaveIcon}
  ondismiss={onKeepEditing}
>
  {#snippet actions()}
    <Button
      variant="outline"
      disabled={busy}
      data-dialog-initial-focus
      onclick={onKeepEditing}>Keep editing</Button
    >
    <Button variant={discardVariant} disabled={busy} onclick={onDiscard}
      >{discardLabel}</Button
    >
    <Button disabled={busy} onclick={onSave}>
      <ButtonIcon icon={SaveIcon} {busy} />
      {busy ? "Saving…" : saveLabel}
    </Button>
  {/snippet}
</ActionDialog>
