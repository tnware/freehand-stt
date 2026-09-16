<script lang="ts">
  import { onDestroy } from "svelte";
  import { CancellablePromise } from "@wailsio/runtime";
  import type { Settings } from "$lib/state";
  import { Session } from "$lib/stores/session.svelte";
  import {
    settings,
    idle,
    serviceWithStatus,
  } from "$lib/stores/session-fixtures-data";
  import { Button } from "$lib/components/ui/button";
  import PendingChangesDialog from "$lib/components/settings/PendingChangesDialog.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";

  const params = new URLSearchParams(location.search);
  const message = params.has("long")
    ? "Unavailable ".repeat(150)
    : "The settings database could not be loaded.";
  const calls: string[] = [];
  let pending:
    | { resolve: (value: Settings) => void; reject: (reason: Error) => void }
    | undefined;
  function recover(action: string) {
    calls.push(action);
    return new CancellablePromise<Settings>((resolve, reject) => {
      pending = { resolve, reject };
    });
  }
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      settings: {
        RetryConfiguration: () => recover("retry"),
        ResetConfiguration: () => recover("reset"),
      },
    }),
  );
  let open = $state(false);
  let busy = $state(false);
  let error = $state("");
  const control = {
    calls,
    complete: (success: boolean) => {
      if (pending) {
        const request = pending;
        pending = undefined;
        if (success) request.resolve(structuredClone(settings));
        else request.reject(new Error("Fixture recovery failed. Try again."));
      } else {
        busy = false;
        if (success) open = false;
        else error = "Fixture save failed. Try again.";
      }
    },
  };
  Object.assign(window, { testDialogs: control });
  onDestroy(() => session.dispose());
</script>

<main class="flex h-screen items-start gap-3 bg-background p-6 text-foreground">
  <Button
    onclick={() => {
      open = true;
      error = "";
    }}>Open unsaved changes</Button
  >
  <Button
    onclick={() =>
      session.editor.applySettingsSnapshot({
        ...settings,
        configuration: {
          recoveryRequired: true,
          errorKind: "database_corrupt",
          message,
        },
      })}>Open recovery</Button
  >
  <PendingChangesDialog
    {open}
    {busy}
    {error}
    title="Save settings changes?"
    description="Save your changes before leaving, or discard them to keep the existing settings."
    onKeepEditing={() => (open = false)}
    onDiscard={() => (open = false)}
    onSave={() => {
      if (!busy) {
        calls.push("save");
        busy = true;
        error = "";
      }
    }}
  />
  <ConfigurationRecoveryDialog {session} />
</main>
