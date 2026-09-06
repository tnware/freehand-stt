<script lang="ts">
  import { CancellablePromise } from "@wailsio/runtime";
  import { Action, Purpose } from "$bindings/savedconnection";
  import { AuthenticationMode } from "$bindings/config";
  import { Session } from "$lib/stores/session.svelte";
  import { settings, idle, serviceWithStatus } from "$lib/stores/session-fixtures-data";
  import SettingsScreen from "$lib/components/settings/SettingsScreen.svelte";
  import type { SettingsSectionID } from "$lib/navigation";
  import { controlledSaves } from "./save-control";

  let current = structuredClone(settings);
  current.savedConnections = {
    entries: [
      {
        id: "original",
        name: "Original server",
        uses: [Purpose.Transcription],
        hasCredential: false,
        details: {
          compatibilityProfile: current.compatibilityProfile,
          baseURL: current.baseURL,
          allowInsecureHTTP: false,
          authenticationMode: AuthenticationMode.AuthenticationModeNone,
          healthPath: "",
          headers: {},
        },
      },
    ],
    selected: { [Purpose.Transcription]: "original" },
  };
  const saves = controlledSaves((request) => {
    const next = { ...current, ...request.settings };
    const change = request.connectionChange;
    if (change) {
      if (change.action !== Action.Create || !change.details)
        throw new Error("Unexpected connection action");
      const id = "created";
      next.savedConnections = {
        entries: [
          ...(current.savedConnections.entries ?? []),
          {
            id,
            name: change.name,
            uses: change.uses ?? [],
            details: change.details,
            hasCredential: false,
          },
        ],
        selected: { ...current.savedConnections.selected, [Purpose.Transcription]: id },
      };
    }
    current = structuredClone(next);
    return structuredClone(current);
  });
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      settings: {
        SaveSettings: (request) => saves.save(structuredClone($state.snapshot(request))),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  window.testSaves = saves.control;
  let active = $state<SettingsSectionID>("server");
</script>

<div class="flex h-screen flex-col text-foreground">
  <SettingsScreen
    {session}
    bind:active
    onClose={() => {}}
    overlayPreviewing={false}
    onStartOverlayPreview={() => {}}
    onStopOverlayPreview={() => {}}
  />
</div>
