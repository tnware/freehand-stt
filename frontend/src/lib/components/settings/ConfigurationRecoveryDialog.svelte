<script lang="ts">
  import { platformPresentation } from "$lib/platform";
  import FileWarningIcon from "@lucide/svelte/icons/file-warning";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RefreshCcwIcon from "@lucide/svelte/icons/refresh-ccw";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import ActionDialog from "$lib/components/common/ActionDialog.svelte";
  import type { Session } from "$lib/stores/session.svelte";

  let { session }: { session: Session } = $props();

  const configuration = $derived(session.editor.applied?.configuration);
  const recoveryRequired = $derived(configuration?.recoveryRequired ?? false);
  const busy = $derived(
    session.editor.configurationRetrying ||
      session.editor.configurationResetting,
  );
  const native = $derived(
    platformPresentation(session.editor.applied?.platform),
  );
</script>

<ActionDialog
  open={recoveryRequired}
  {busy}
  dismissible={false}
  icon={FileWarningIcon}
  tone="danger"
  title="Saved settings need attention"
  description="Transcription and settings changes are paused until your saved configuration can be loaded or you choose to reset it."
  error={session.messages.error}
>
  <Alert.Root variant="destructive">
    <Alert.Title>Configuration could not be loaded</Alert.Title>
    <Alert.Description
      >{configuration?.message ??
        "The saved configuration is invalid."}</Alert.Description
    >
  </Alert.Root>
  <p class="text-xs leading-relaxed text-muted-foreground">
    Retry after fixing file access or restoring a current-version backup with
    Freehand closed. Reset archives the existing database and starts with safe
    defaults. Credentials in
    {native.credentialStore} are kept; reconfigure connections and enter API keys
    again.
  </p>
  {#snippet actions()}
    <Button
      variant="destructive"
      disabled={busy}
      onclick={() => session.editor.resetConfiguration()}
    >
      {#if session.editor.configurationResetting}
        <LoaderCircleIcon
          data-icon="inline-start"
          class="motion-safe:animate-spin"
        />
      {:else}<RotateCcwIcon data-icon="inline-start" />{/if}
      {session.editor.configurationResetting
        ? "Resetting…"
        : "Reset to defaults"}
    </Button>
    <Button
      disabled={busy}
      data-dialog-initial-focus
      onclick={() => session.editor.retryConfiguration()}
    >
      {#if session.editor.configurationRetrying}
        <LoaderCircleIcon
          data-icon="inline-start"
          class="motion-safe:animate-spin"
        />
      {:else}<RefreshCcwIcon data-icon="inline-start" />{/if}
      {session.editor.configurationRetrying ? "Loading…" : "Retry loading"}
    </Button>
  {/snippet}
</ActionDialog>
