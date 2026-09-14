<script lang="ts">
  import DownloadIcon from "@lucide/svelte/icons/download";
  import { Button } from "$lib/components/ui/button";
  import Chain from "./Chain.svelte";
  import Stage, { type StageTone } from "./Stage.svelte";
  import StagePick from "./StagePick.svelte";
  import TranscribeStage from "./TranscribeStage.svelte";
  import CleanupStage from "./CleanupStage.svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { endpointHost } from "$lib/utils/endpoint";
  import { taskConnectionStatus } from "$lib/utils/connection";
  import { TTSPhase } from "$lib/state";
  import type { Purpose } from "$bindings/savedconnection";
  import SpeechQuickSettings from "$lib/components/home/SpeechQuickSettings.svelte";

  let {
    session,
    now,
    onOpenSpeech,
    onOpenRuntimes,
    onAddConnection,
  }: {
    session: Session;
    now: number;
    onOpenSpeech: () => void;
    onOpenRuntimes: () => void;
    onAddConnection: (purpose: Purpose) => void;
  } = $props();

  const status = $derived(session.speech.status);
  const settings = $derived(session.editor.applied ?? session.editor.draft);
  const speech = $derived(settings?.textToSpeech);

  const connection = $derived(taskConnectionStatus("tts", session.editor, now));
  const generating = $derived(status.phase === TTSPhase.Generating);
  const failed = $derived(status.phase === TTSPhase.Failed);
  const playing = $derived(
    status.phase === TTSPhase.Playing || status.phase === TTSPhase.Paused,
  );

  const synthesizeTone = $derived<StageTone>(
    failed
      ? "bad"
      : generating
        ? "busy"
        : connection.dot === "bg-success"
          ? "ok"
          : connection.dot === "bg-warning"
            ? "warn"
            : connection.dot === "bg-destructive"
              ? "bad"
              : "idle",
  );

  const characters = $derived(session.speech.draft.length);
</script>

<!--
  The same four positions running the other way: text in, audio out. Cleanup
  has nothing to act on here, so it dims in place rather than disappearing —
  otherwise stage 04 would mean "deliver" in two chains and "clean up" in a
  third, and the shape would stop being worth learning.
-->
<Chain
  label="Text to speech chain"
  links={[
    { label: "text", tone: generating ? "live" : "idle" },
    { label: "audio", tone: "idle" },
    { label: "audio", tone: playing ? "live" : "idle" },
  ]}
  stages={[source, synthesize, clean, deliver]}
/>

{#snippet source()}
  <Stage ordinal="01" label="Source" tone={characters > 0 ? "ok" : "idle"}>
    <StagePick
      value="Text you type"
      meta={`${characters.toLocaleString()} character${characters === 1 ? "" : "s"}`}
      label="Speech source"
      onOpen={onOpenSpeech}
    />
    {#snippet footer()}
      <span
        class="rounded-sm border border-border px-1.5 py-0.5 text-[11px] text-muted-foreground"
        >Composer below</span
      >
    {/snippet}
  </Stage>
{/snippet}

{#snippet synthesize()}
  <TranscribeStage
    ordinal="02"
    label="Synthesize"
    model={speech?.model ?? ""}
    source={speech?.baseURL ? endpointHost(speech.baseURL) : ""}
    tone={synthesizeTone}
    state={speech?.enabled === false ? "Off" : connection.label}
    notice={failed ? (status.message ?? "") : ""}
    onOpen={onOpenSpeech}
    panel={settings ? speechPanel : undefined}
  />
{/snippet}

{#snippet speechPanel()}
  <div class="p-3">
    <SpeechQuickSettings
      settings={settings!}
      runtime={session.runtime}
      onManageRuntime={onOpenRuntimes}
      editor={session.editor}
      disabled={session.editor.saving}
      {onAddConnection}
      onOpenSettings={onOpenSpeech}
    />
  </div>
{/snippet}

{#snippet clean()}
  <CleanupStage
    unused
    enabled={false}
    model=""
    onToggle={() => {}}
    onOpen={onOpenSpeech}
  />
{/snippet}

{#snippet deliver()}
  <Stage
    ordinal="04"
    label="Deliver"
    tone={failed ? "bad" : playing || status.canSave ? "ok" : "idle"}
  >
    <StagePick
      value="Play or save"
      meta="audio file"
      label="Speech delivery"
      onOpen={onOpenSpeech}
    />
    {#snippet footer()}
      {#if status.canSave}
        <Button
          size="xs"
          onclick={() => void session.speech.saveTTSAudio()}
          disabled={session.speech.saving}
        >
          <DownloadIcon class="size-3" />
          Save
        </Button>
      {:else}
        <span></span>
      {/if}
    {/snippet}
  </Stage>
{/snippet}
