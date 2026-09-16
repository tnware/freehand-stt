<script lang="ts">
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import BuiltInConnectionDetails from "$lib/components/settings/BuiltInConnectionDetails.svelte";
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import RuntimeModelCatalog from "$lib/components/runtimes/RuntimeModelCatalog.svelte";
  import FileBar from "$lib/components/home/FileBar.svelte";
  import { CheckKind, CheckStatus } from "$bindings/connection";
  import { Purpose, type Connection } from "$bindings/savedconnection";
  import { FileTranscriptionPhase } from "$lib/state";
  import {
    connectionResult,
    settings,
  } from "$lib/stores/session-fixtures-data";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { createRuntimeFixture } from "./runtime-fixture";

  const token = "abcdefghijklmnopqrstuvwxyz0123456789".repeat(5);
  const filename = `${token}.wav`;
  const params = new URLSearchParams(location.search);
  const paneWidth = params.get("pane") === "wide" ? 520 : 280;
  const fixture = createRuntimeFixture(true, () => {});
  const provider = fixture.providers[0];
  const models = provider.models!.map((model) => ({
    ...model,
    name: token,
    description: token,
  }));
  const row = fixture.control.snapshot()[0];
  const connection: Connection = {
    id: "long-content",
    name: token,
    builtIn: true,
    hasCredential: false,
    uses: [Purpose.Voice, Purpose.Transcription],
    details: {
      managedInstanceID: row.instance.id,
      compatibilityProfile: settings.compatibilityProfile,
      baseURL: "http://127.0.0.1:8080/v1",
      allowInsecureHTTP: false,
      authenticationMode: settings.authenticationMode,
      healthPath: "",
      headers: {},
    },
  };
  let action = $state("");
  const noop = () => {};
</script>

<main
  class="mx-auto h-screen max-w-full space-y-6 overflow-y-auto bg-background p-3 text-foreground"
  style:width={`${paneWidth}px`}
>
  <section aria-label="Long connection diagnostics">
    <ConnectionDiagnostics
      result={{
        ...connectionResult,
        serverVersion: token,
        checks: [
          {
            kind: CheckKind.CheckConnection,
            status: CheckStatus.CheckAttention,
            summary: token,
            detail: `${token}\nCheck your server configuration.`,
          },
        ],
      }}
      onCheck={() => (action = "Checked")}
    />
  </section>
  <section aria-label="Long built-in details">
    <BuiltInConnectionDetails
      {connection}
      instance={row.instance}
      status={row}
      providers={[{ ...provider, models }]}
      onManageRuntime={() => (action = "Runtime opened")}
      onWorkflow={noop}
    />
  </section>
  <section aria-label="Long model catalog">
    <RuntimeModelCatalog
      {models}
      entry={{ ...provider, models }}
      {row}
      installed
      locked={false}
      busy={false}
      cancelling={false}
      view={runtimePresentation(row.status)}
      onRefresh={noop}
      onGet={() => (action = "Download requested")}
      onSelect={() => (action = "Model selected")}
      onRemove={noop}
      onCancel={noop}
      onDisableSpeech={noop}
    />
  </section>
  <section aria-label="Long filename">
    <FileBar
      status={{
        generation: 1,
        phase: FileTranscriptionPhase.FileTranscriptionFailed,
        fileName: filename,
        fileSize: 10240,
        streaming: false,
        buffered: false,
        streamingUnavailable: false,
        streamingProfileUnavailable: false,
        transcriptRevision: 0,
        canStart: true,
        canCancel: false,
        canCopy: false,
        message: token.repeat(12),
      }}
      onChoose={() => (action = "File chooser opened")}
      onStart={() => (action = "Retry requested")}
      onClear={() => (action = "File cleared")}
      onCancel={noop}
      onTryStreamingAgain={noop}
      onOpenSettings={() => (action = "File settings opened")}
      showSettings={false}
    />
  </section>
  <FeedbackDetails
    title="Long error message"
    message={token.repeat(12)}
    actionLabel="Open settings"
    onAction={() => (action = "Settings opened")}
  />
  <p role="status">{action}</p>
</main>
