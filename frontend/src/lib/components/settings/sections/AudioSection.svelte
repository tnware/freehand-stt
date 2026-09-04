<script lang="ts">
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { Label } from "$lib/components/ui/label";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import * as Select from "$lib/components/ui/select";
  import * as Slider from "$lib/components/ui/slider";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import { VADMode, type Device, type Settings } from "$lib/state";
  import {
    SYSTEM_DEFAULT_LABEL,
    SYSTEM_DEFAULT_MICROPHONE,
    microphoneLabel,
    microphoneMissing,
  } from "$lib/utils/microphone";

  let {
    settings = $bindable(),
    devices,
    microphoneChoice,
    busy = false,
    onChooseMicrophone,
    onRefreshDevices,
  }: {
    settings: Settings;
    devices: Device[];
    microphoneChoice: string;
    busy?: boolean;
    onChooseMicrophone: (choice: string) => void;
    onRefreshDevices: () => void;
  } = $props();

  const VAD_MODES = [
    {
      value: VADMode.VADModeQuality,
      label: "Quality",
      description: "Keeps more quiet or distant speech.",
    },
    {
      value: VADMode.VADModeLowBitrate,
      label: "Low bitrate",
      description: "Tuned for compressed, phone-like audio.",
    },
    {
      value: VADMode.VADModeAggressive,
      label: "Aggressive",
      description: "Balanced default for everyday dictation.",
    },
    {
      value: VADMode.VADModeVeryAggressive,
      label: "Very aggressive",
      description: "Rejects the most noise; may miss soft speech.",
    },
  ] as const;

  const label = $derived(microphoneLabel(microphoneChoice, devices));
  const missing = $derived(microphoneMissing(microphoneChoice, devices));
  const maximumDuration = $derived(settings.silenceSplitting ? 3600 : 262);

  function toggleVAD(enabled: boolean) {
    settings.vadEnabled = enabled;
    if (!enabled) {
      settings.silenceTrimming = false;
      settings.autoStopEnabled = false;
      toggleSilenceSplitting(false);
    }
  }

  function toggleSilenceTrimming(enabled: boolean) {
    if (enabled) settings.vadEnabled = true;
    settings.silenceTrimming = enabled;
  }

  function toggleAutoStop(enabled: boolean) {
    if (enabled) settings.vadEnabled = true;
    if (enabled && settings.autoStopSilenceMilliseconds < settings.vadActivitySilenceMilliseconds) {
      settings.autoStopSilenceMilliseconds = settings.vadActivitySilenceMilliseconds;
    }
    settings.autoStopEnabled = enabled;
  }

  function changeActivitySilence(value: number) {
    settings.vadActivitySilenceMilliseconds = value;
    if (settings.autoStopEnabled && settings.autoStopSilenceMilliseconds < value) {
      settings.autoStopSilenceMilliseconds = value;
    }
  }

  function toggleSilenceSplitting(enabled: boolean) {
    if (enabled) settings.vadEnabled = true;
    settings.silenceSplitting = enabled;
    if (!enabled && settings.maxDurationSeconds > 262) settings.maxDurationSeconds = 262;
  }

  function chooseVADMode(value: string) {
    settings.vadMode = value as VADMode;
  }

  function seconds(milliseconds: number) {
    return `${(milliseconds / 1000).toFixed(2).replace(/\.?0+$/, "")} s`;
  }
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <ValueRow
      id="microphone-select"
      label="Microphone"
      hint={missing
        ? "This device is not connected right now. Reconnect it or choose System default; the saved choice is kept."
        : ""}
    >
      {#snippet control()}
        <Select.Root type="single" value={microphoneChoice} onValueChange={onChooseMicrophone}>
          <Select.Trigger
            id="microphone-select"
            class="h-auto w-full border-0 bg-transparent p-0 text-[15px] shadow-none focus-visible:ring-0"
          >
            <span class="flex min-w-0 items-center gap-2">
              {#if missing}
                <TriangleAlertIcon class="size-4 shrink-0 text-muted-foreground" />
              {/if}
              <span class="truncate">{label}</span>
            </span>
          </Select.Trigger>
          <Select.Content>
            <Select.Group>
              <Select.Item value={SYSTEM_DEFAULT_MICROPHONE}>{SYSTEM_DEFAULT_LABEL}</Select.Item>
              {#each devices as device (device.id)}
                <Select.Item value={device.id}>{device.name}</Select.Item>
              {/each}
            </Select.Group>
          </Select.Content>
        </Select.Root>
      {/snippet}
      {#snippet action()}
        <Button variant="secondary" size="sm" disabled={busy} onclick={onRefreshDevices}>
          <RefreshCwIcon data-icon="inline-start" class={busy ? "animate-spin" : ""} />
          Refresh
        </Button>
      {/snippet}
    </ValueRow>

    <ValueRow
      id="max-duration"
      label="Maximum duration"
      hint={settings.silenceSplitting
        ? "Total recording limit. Completed segments are transcribed in order while recording continues."
        : "Recording stops here, and anything said afterwards is not captured. The 262-second ceiling fits in one 16 kHz request."}
    >
      {#snippet control()}
        <div class="flex items-baseline gap-2">
          <ValueInput
            id="max-duration"
            type="number"
            min="1"
            max={maximumDuration}
            class="w-20"
            bind:value={settings.maxDurationSeconds}
          />
          <span class="font-mono text-sm text-muted-foreground">seconds</span>
        </div>
      {/snippet}
    </ValueRow>
  </SettingsCard>

  <SettingsCard>
    <SettingRow
      title="Voice activity detection"
      description="Make local WebRTC speech detection available to the features below. The detector runs only while a dependent feature needs it, analyzes fixed 20 ms frames, and sends no audio anywhere itself."
    >
      {#snippet control()}
        <Switch
          id="vad-enabled"
          checked={settings.vadEnabled}
          onCheckedChange={toggleVAD}
          aria-label="Enable voice activity detection"
        />
      {/snippet}
    </SettingRow>

    {#if settings.vadEnabled}
      <ValueRow
        id="vad-engine"
        label="Detector engine"
        hint="Pinned native libfvad engine. It needs no model download, background service, or network request."
      >
        {#snippet control()}
          <span id="vad-engine" class="text-[15px]">WebRTC VAD (libfvad)</span>
        {/snippet}
      </ValueRow>

      <SettingRow
        title="Detection mode"
        description="Choose how readily background sound is rejected. Aggressive is the practical default."
      >
        <RadioGroup.Root
          class="grid-cols-1 gap-2 sm:grid-cols-2"
          value={settings.vadMode}
          onValueChange={chooseVADMode}
          aria-label="Voice activity detection mode"
        >
          {#each VAD_MODES as mode (mode.value)}
            <Label
              for={`vad-mode-${mode.value}`}
              class="flex cursor-pointer items-start gap-3 rounded-lg border border-hairline bg-background/35 px-3 py-2.5 transition-colors hover:bg-accent/55"
            >
              <RadioGroup.Item
                id={`vad-mode-${mode.value}`}
                value={mode.value}
                class="mt-0.5"
              />
              <span class="min-w-0">
                <span class="block text-xs font-medium text-foreground">{mode.label}</span>
                <span class="mt-0.5 block text-[11px] leading-relaxed text-muted-foreground">
                  {mode.description}
                </span>
              </span>
            </Label>
          {/each}
        </RadioGroup.Root>
      </SettingRow>

      <SettingRow
        title="Silence indicator delay"
        description="Require a continuous quiet gap before the live microphone indicator changes to silence. More delay reduces flicker between words."
      >
        {#snippet control()}
          <Badge variant="secondary">{settings.vadActivitySilenceMilliseconds} ms</Badge>
        {/snippet}
        <Slider.Root
          id="vad-activity-silence"
          type="single"
          min={100}
          max={1500}
          step={100}
          value={settings.vadActivitySilenceMilliseconds}
          onValueChange={changeActivitySilence}
          aria-label="Silence indicator delay"
        />
        <div class="mt-2 flex justify-between text-[10px] text-muted-foreground">
          <span>Responsive</span>
          <span>Steady</span>
        </div>
      </SettingRow>
    {/if}
  </SettingsCard>

  {#if settings.vadEnabled}
    <SettingsCard>
      <SettingRow
        title="Trim silence"
        description="Remove quiet audio before the first detected phrase and after the last one before the request is created."
      >
        {#snippet control()}
          <Switch
            id="silence-trimming"
            checked={settings.silenceTrimming}
            onCheckedChange={toggleSilenceTrimming}
            aria-label="Trim leading and trailing silence"
          />
        {/snippet}
      </SettingRow>

      {#if settings.silenceTrimming}
        <SettingRow
          title="Speech padding"
          description="Keep this much context on both sides of detected speech so breaths and word edges are not clipped."
        >
          {#snippet control()}
            <Badge variant="secondary">{settings.speechPaddingMilliseconds} ms</Badge>
          {/snippet}
          <Slider.Root
            id="speech-padding"
            type="single"
            min={0}
            max={1000}
            step={50}
            value={settings.speechPaddingMilliseconds}
            onValueChange={(value) => (settings.speechPaddingMilliseconds = value)}
            aria-label="Speech padding"
          />
          <div class="mt-2 flex justify-between text-[10px] text-muted-foreground">
            <span>Tighter</span>
            <span>More context</span>
          </div>
        </SettingRow>
      {/if}

      <SettingRow
        title="Stop after I finish speaking"
        description="End recording after confirmed speech is followed by a sustained pause. It never stops merely because you have not started speaking yet."
      >
        {#snippet control()}
          <Switch
            id="automatic-stop"
            checked={settings.autoStopEnabled}
            onCheckedChange={toggleAutoStop}
            aria-label="Stop recording after speech ends"
          />
        {/snippet}
      </SettingRow>

      {#if settings.autoStopEnabled}
        <SettingRow
          title="Pause before stopping"
          description="How long silence must continue after speech before recording ends. Speaking again cancels the countdown immediately."
        >
          {#snippet control()}
            <Badge variant="secondary">{seconds(settings.autoStopSilenceMilliseconds)}</Badge>
          {/snippet}
          <Slider.Root
            id="automatic-stop-silence"
            type="single"
            min={Math.max(500, settings.vadActivitySilenceMilliseconds)}
            max={10000}
            step={250}
            value={settings.autoStopSilenceMilliseconds}
            onValueChange={(value) => (settings.autoStopSilenceMilliseconds = value)}
            aria-label="Pause before automatic stop"
          />
          <div class="mt-2 flex justify-between text-[10px] text-muted-foreground">
            <span>0.5 s</span>
            <span>10 s</span>
          </div>
        </SettingRow>

        <SettingRow
          title="Speech required to arm"
          description="Require this much cumulative detected speech first. This prevents a tap, cough, or brief noise from starting an automatic-stop countdown."
        >
          {#snippet control()}
            <Badge variant="secondary">{settings.autoStopMinimumSpeechMilliseconds} ms</Badge>
          {/snippet}
          <Slider.Root
            id="automatic-stop-minimum-speech"
            type="single"
            min={100}
            max={5000}
            step={100}
            value={settings.autoStopMinimumSpeechMilliseconds}
            onValueChange={(value) => (settings.autoStopMinimumSpeechMilliseconds = value)}
            aria-label="Speech required to arm automatic stop"
          />
          <div class="mt-2 flex justify-between text-[10px] text-muted-foreground">
            <span>100 ms</span>
            <span>5 s</span>
          </div>
        </SettingRow>
      {/if}
    </SettingsCard>

    <SettingsCard>
      <SettingRow
        title="Split long dictation on silence"
        description="After the preferred length, the next sustained pause closes and transcribes a segment. Requests stay ordered and insertion still happens once."
      >
        {#snippet control()}
          <Switch
            id="silence-splitting"
            checked={settings.silenceSplitting}
            onCheckedChange={toggleSilenceSplitting}
            aria-label="Split long dictation on silence"
          />
        {/snippet}
      </SettingRow>

      {#if settings.silenceSplitting}
        <SettingRow
          title="Preferred segment length"
          description="The next qualified pause after this point closes the segment. A 240-second safety ceiling applies if no pause arrives."
        >
          {#snippet control()}
            <Badge variant="secondary">{settings.segmentSeconds} s</Badge>
          {/snippet}
          <Slider.Root
            id="segment-duration"
            type="single"
            min={15}
            max={180}
            step={15}
            value={settings.segmentSeconds}
            onValueChange={(value) => (settings.segmentSeconds = value)}
            aria-label="Preferred segment length"
          />
        </SettingRow>

        <SettingRow
          title="Pause required to split"
          description="This boundary is independent of automatic stop. Shorter pauses checkpoint more eagerly; 700 ms is the practical default."
        >
          {#snippet control()}
            <Badge variant="secondary">{settings.segmentSilenceMilliseconds} ms</Badge>
          {/snippet}
          <Slider.Root
            id="segment-silence"
            type="single"
            min={200}
            max={3000}
            step={100}
            value={settings.segmentSilenceMilliseconds}
            onValueChange={(value) => (settings.segmentSilenceMilliseconds = value)}
            aria-label="Pause required to split"
          />
        </SettingRow>
      {/if}
    </SettingsCard>

    <SettingsCard>
      <SettingRow
        title="How the speech policy works"
        description="The detector runs locally on fixed 20 ms frames. Two speech frames confirm a phrase; the selected indicator delay smooths the visual state. Trimming keeps your chosen padding, automatic stop waits for both the minimum speech and full pause, and segment splitting uses its own later boundary. Manual stop and the maximum duration always remain available."
      />
    </SettingsCard>
  {/if}
</div>
