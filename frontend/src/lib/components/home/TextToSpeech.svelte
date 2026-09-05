<script lang="ts">
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import SparklesIcon from "@lucide/svelte/icons/sparkles";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import TransportShell from "$lib/components/home/TransportShell.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import { TTSPhase, TTSSource, type Settings, type TTSStatus } from "$lib/state";
  import { cn } from "$lib/utils";

  const maximumCharacters = 4096;

  let {
    settings,
    status,
    unavailable = false,
    onSpeak,
    onPause,
    onResume,
    onRestart,
    onStop,
    onSave,
    onClear,
    onOpenSettings,
  }: {
    settings: Settings["textToSpeech"];
    status: TTSStatus;
    unavailable?: boolean;
    onSpeak: (text: string) => void;
    onPause: () => void;
    onResume: () => void;
    onRestart: () => void;
    onStop: () => void;
    onSave: () => void;
    onClear: () => void;
    onOpenSettings: () => void;
  } = $props();

  let text = $state("");
  const characterCount = $derived(Array.from(text).length);
  const isOwnSession = $derived(status.source === TTSSource.SourceCompose);
  const working = $derived(
    isOwnSession &&
      (status.phase === TTSPhase.Generating ||
        status.phase === TTSPhase.Playing ||
        status.phase === TTSPhase.Paused),
  );
  const showPlayback = $derived(
    isOwnSession && status.phase !== TTSPhase.Idle && status.phase !== TTSPhase.Cancelled,
  );
  const configured = $derived(
    settings.enabled &&
      Boolean(settings.baseURL.trim() && settings.model.trim() && settings.voice.trim()),
  );
  const canSpeak = $derived(
    configured &&
      !unavailable &&
      !working &&
      characterCount > 0 &&
      characterCount <= maximumCharacters,
  );
  const stateLabel = $derived.by(() => {
    if (!configured) return "Setup needed";
    if (!isOwnSession) return "Ready";
    if (status.phase === TTSPhase.Generating) return "Generating";
    if (status.phase === TTSPhase.Playing) return "Speaking";
    if (status.phase === TTSPhase.Paused) return "Paused";
    if (status.phase === TTSPhase.Completed) return "Complete";
    if (status.phase === TTSPhase.Failed) return "Failed";
    return "Ready";
  });

  function compactLabel(value: string, fallback: string): string {
    const trimmed = value.trim();
    return trimmed ? (trimmed.split("/").at(-1) ?? trimmed) : fallback;
  }

  const failed = $derived(isOwnSession && status.phase === TTSPhase.Failed);
  const rail = $derived(
    failed
      ? "error"
      : isOwnSession && status.phase === TTSPhase.Completed
        ? "done"
        : isOwnSession && status.phase === TTSPhase.Generating
          ? "working"
          : "hidden",
  );
</script>

<div class="flex shrink-0 flex-col">
  <TransportShell {rail} tall stageGrid={false} busy={working} state={status.phase}>
    {#snippet control()}
      <span
        class={cn(
          "grid size-[62px] place-items-center self-start rounded-full border",
          configured
            ? "border-accent-edge bg-accent-wash text-accent-text"
            : "border-hairline bg-control-fill text-muted-foreground",
        )}
        style="margin-top: 1.25rem"
        aria-hidden="true"
      >
        {#if working && status.phase === TTSPhase.Generating}
          <LoaderCircleIcon class="size-[24px] animate-spin motion-reduce:animate-none" />
        {:else}
          <Volume2Icon class="size-[24px]" />
        {/if}
      </span>
    {/snippet}

    {#snippet stage()}
      <div class="tts-stage flex h-full flex-col gap-2.5 py-4">
        <div class="flex items-center gap-2.5">
          <h2 class="caption">Write something to speak</h2>
          <span class="flex-1"></span>
          <span
            class="figure shrink-0 truncate text-[10px] text-ink-quiet"
            title={`${settings.model || "No model"} · ${settings.voice || "No voice"}`}
          >
            {compactLabel(settings.model, "model")} · {compactLabel(settings.voice, "voice")}
          </span>
        </div>

        <Textarea
          bind:value={text}
          maxlength={maximumCharacters}
          disabled={!configured || unavailable || working}
          class="field-sizing-fixed min-h-0 flex-1 resize-none overflow-y-auto bg-well text-[13px] leading-relaxed"
          placeholder="Enter text for Freehand to read aloud…"
          aria-label="Text to speak"
        />
      </div>
    {/snippet}

    {#snippet readout()}
      <div class="tts-readout flex h-full flex-col gap-2.5 py-4">
        <div class="tts-endpoint flex min-w-0 flex-col gap-2.5">
          <div class="flex items-center gap-2">
            <ProviderIcon profile={settings.compatibilityProfile} size={20} />
            <h2 class="caption">Endpoint</h2>
          </div>
          <span class="figure truncate text-[11px] text-card-foreground" title={settings.baseURL}>
            {settings.baseURL || "Not configured"}
          </span>
        </div>
        <div class="tts-health flex items-center gap-2">
          <span
            class={cn(
              "size-[7px] rounded-full",
              failed
                ? "bg-destructive"
                : working
                  ? "bg-primary shadow-[0_0_8px_var(--primary)]"
                  : configured
                    ? "bg-success"
                    : "bg-border",
            )}
          ></span>
          <span class="caption text-secondary-foreground">{stateLabel}</span>
        </div>
        <p class="tts-note figure mt-auto text-[10px] leading-relaxed text-ink-quiet">
          {configured
            ? "Audio is generated only when you press Speak."
            : "Configure a speech endpoint, model, and voice first."}
        </p>
        <div class="tts-actions flex items-center gap-2">
          <span class="figure mr-auto text-[10px] text-ink-quiet">
            {characterCount.toLocaleString()} / {maximumCharacters.toLocaleString()}
          </span>
          {#if !configured}
            <Button
              variant="outline"
              size="sm"
              class="h-[26px] px-2.5 text-[11.5px]"
              onclick={onOpenSettings}
            >
              <SettingsIcon class="size-3" />
              Speech settings
            </Button>
          {:else}
            <Button
              variant="ghost"
              size="sm"
              class="h-[26px] px-2.5 text-[11.5px]"
              disabled={!text || working}
              onclick={() => (text = "")}
            >
              Clear
            </Button>
            <Button
              size="sm"
              class="h-[26px] px-3 text-[11.5px]"
              disabled={!canSpeak}
              onclick={() => onSpeak(text)}
            >
              <SparklesIcon class="size-3" />
              Speak
            </Button>
          {/if}
        </div>
      </div>
    {/snippet}
  </TransportShell>

  {#if showPlayback}
    <PlaybackBar {status} {onPause} {onResume} {onRestart} {onStop} {onSave} {onClear} />
  {/if}
</div>

<style>
  /* The narrow transport gives endpoint facts one compact row instead of a
     second tall panel beneath the editor. The same information remains
     visible, but switching to Text to speech moves the workspace far less. */
  @container (max-width: 699px) {
    .tts-stage {
      padding-block: 0;
    }
    .tts-readout {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      grid-template-areas:
        "endpoint health"
        "actions actions";
      align-content: center;
      column-gap: 1rem;
      row-gap: 0.25rem;
      padding-block: 0;
    }
    .tts-endpoint {
      grid-area: endpoint;
      display: flex;
      flex-direction: row;
      align-items: center;
      gap: 0;
    }
    .tts-endpoint :global(h2) {
      display: none;
    }
    .tts-health {
      grid-area: health;
    }
    .tts-note {
      display: none;
    }
    .tts-actions {
      grid-area: actions;
    }
    .tts-actions :global(button) {
      height: 1.5rem;
      padding-inline: 0.5rem;
    }
  }
</style>
