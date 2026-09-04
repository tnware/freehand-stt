<script lang="ts">
  import FileWarningIcon from "@lucide/svelte/icons/file-warning";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RefreshCcwIcon from "@lucide/svelte/icons/refresh-ccw";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";
  import type { Session } from "$lib/stores/session.svelte";

  let { session }: { session: Session } = $props();

  const configuration = $derived(session.appliedSettings?.configuration);
  const recoveryRequired = $derived(configuration?.recoveryRequired ?? false);
</script>

<Dialog.Root open={recoveryRequired} onOpenChange={() => {}}>
  <Dialog.Content
    showCloseButton={false}
    escapeKeydownBehavior="ignore"
    interactOutsideBehavior="ignore"
    class="gap-0 bg-dialog-surface p-0 shadow-xl ring-dialog-stroke sm:max-w-[460px]"
  >
    <Dialog.Header class="border-b border-hairline px-5 py-4">
      <div class="flex items-start gap-3">
        <div class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-destructive/10 text-destructive">
          <FileWarningIcon class="size-5" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <Dialog.Title class="text-base font-semibold">Saved settings need attention</Dialog.Title>
          <Dialog.Description class="mt-1 text-[13px] leading-relaxed">
            Freehand did not replace your configuration with defaults. Transcription and settings
            changes are paused until the saved file can be loaded or you explicitly reset it.
          </Dialog.Description>
        </div>
      </div>
    </Dialog.Header>

    <div class="space-y-3 px-5 py-4">
      <Alert.Root variant="destructive">
        <Alert.Title>Configuration could not be loaded</Alert.Title>
        <Alert.Description>{configuration?.message ?? "The saved configuration is invalid."}</Alert.Description>
      </Alert.Root>

      {#if session.error}
        <Alert.Root variant="destructive">
          <Alert.Title>Recovery action failed</Alert.Title>
          <Alert.Description>{session.error}</Alert.Description>
        </Alert.Root>
      {/if}

      <p class="text-xs leading-relaxed text-muted-foreground">
        If you edit or restore the settings file outside Freehand, retry loading it. Resetting
        replaces that file with safe defaults; credentials stored by Windows are not deleted.
      </p>
    </div>

    <Dialog.Footer class="border-t border-hairline bg-layer-fill px-5 py-4 sm:justify-between">
      <Button
        variant="destructive"
        disabled={session.configurationRetrying || session.configurationResetting}
        onclick={() => session.resetConfiguration()}
      >
        {#if session.configurationResetting}
          <LoaderCircleIcon data-icon="inline-start" class="animate-spin motion-reduce:animate-none" />
        {:else}
          <RotateCcwIcon data-icon="inline-start" />
        {/if}
        {session.configurationResetting ? "Resetting…" : "Reset to defaults"}
      </Button>
      <Button
        disabled={session.configurationRetrying || session.configurationResetting}
        onclick={() => session.retryConfiguration()}
      >
        {#if session.configurationRetrying}
          <LoaderCircleIcon data-icon="inline-start" class="animate-spin motion-reduce:animate-none" />
        {:else}
          <RefreshCcwIcon data-icon="inline-start" />
        {/if}
        {session.configurationRetrying ? "Loading…" : "Retry loading"}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
