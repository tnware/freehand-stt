<script lang="ts">
  import EyeIcon from "@lucide/svelte/icons/eye";
  import SquareIcon from "@lucide/svelte/icons/square";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import * as Slider from "$lib/components/ui/slider";
  import { Switch } from "$lib/components/ui/switch";
  import * as ToggleGroup from "$lib/components/ui/toggle-group";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import {
    OverlayAnchor,
    OverlayLayout,
    OverlayMotion,
    OverlaySurface,
    OverlayVisibility,
    OverlayVisualizer,
    type Settings,
  } from "$lib/state";

  let {
    settings = $bindable(),
    previewing,
    canPreview,
    onStartPreview,
    onStopPreview,
  }: {
    settings: Settings;
    previewing: boolean;
    canPreview: boolean;
    onStartPreview: () => void;
    onStopPreview: () => void;
  } = $props();

  const layoutChoices = [
    { value: OverlayLayout.OverlayLayoutMinimal, label: "Minimal", shape: "size-3 rounded-full" },
    { value: OverlayLayout.OverlayLayoutCapsule, label: "Capsule", shape: "h-2.5 w-7 rounded-full" },
    { value: OverlayLayout.OverlayLayoutMeter, label: "Meter", shape: "h-2.5 w-9 rounded-full" },
    { value: OverlayLayout.OverlayLayoutDetailed, label: "Detailed", shape: "h-4 w-8 rounded-[3px]" },
  ];

  const anchorChoices = [
    { value: OverlayAnchor.OverlayAnchorTopLeft, label: "Top left" },
    { value: OverlayAnchor.OverlayAnchorTopCenter, label: "Top center" },
    { value: OverlayAnchor.OverlayAnchorTopRight, label: "Top right" },
    { value: OverlayAnchor.OverlayAnchorBottomLeft, label: "Bottom left" },
    { value: OverlayAnchor.OverlayAnchorBottomCenter, label: "Bottom center" },
    { value: OverlayAnchor.OverlayAnchorBottomRight, label: "Bottom right" },
  ];

  function chooseLayout(value: string) {
    if (value) settings.overlayLayout = value as Settings["overlayLayout"];
  }

  function chooseVisualizer(value: string) {
    if (value) settings.overlayVisualizer = value as Settings["overlayVisualizer"];
  }

  function chooseSurface(value: string) {
    if (value) settings.overlaySurface = value as Settings["overlaySurface"];
  }

  function chooseAnchor(value: string) {
    if (value) settings.overlayAnchor = value as Settings["overlayAnchor"];
  }

  function chooseVisibility(value: string) {
    if (value) settings.overlayVisibility = value as Settings["overlayVisibility"];
  }

  function chooseMotion(value: string) {
    if (value) settings.overlayMotion = value as Settings["overlayMotion"];
  }
</script>

<div class="flex flex-col gap-3.5">
  <SettingsCard>
    <SettingRow
      title="Show status overlay"
      description="Display a passive, click-through status surface above other applications. Turning this off releases the native window completely."
    >
      {#snippet control()}
        <Switch id="overlay-enabled" bind:checked={settings.overlayEnabled} aria-label="Show status overlay" />
      {/snippet}
    </SettingRow>

    <SettingRow
      title="Native preview"
      description="Cycle through recording, silence, countdown, processing, delivery, and failure using the real overlay renderer. Draft changes appear immediately and are not saved."
    >
      {#snippet control()}
        <Button variant={previewing ? "secondary" : "outline"} size="sm" disabled={!canPreview} onclick={previewing ? onStopPreview : onStartPreview}>
          {#if previewing}
            <SquareIcon data-icon="inline-start" />
            Stop preview
          {:else}
            <EyeIcon data-icon="inline-start" />
            Preview overlay
          {/if}
        </Button>
      {/snippet}
    </SettingRow>
  </SettingsCard>

  <SettingsCard>
    <SettingRow title="Layout" description="Choose a curated amount of status detail. Capsule preserves Freehand's original presentation.">
      <ToggleGroup.Root class="grid w-full grid-cols-4" type="single" variant="outline" size="sm" value={settings.overlayLayout} onValueChange={chooseLayout} aria-label="Overlay layout" disabled={!settings.overlayEnabled && !previewing}>
        {#each layoutChoices as choice (choice.value)}
          <ToggleGroup.Item value={choice.value} class="h-11 min-w-0 flex-col gap-1 rounded-none px-1 text-[11px]">
            <span class={`border border-current bg-current/15 ${choice.shape}`} aria-hidden="true"></span>
            {choice.label}
          </ToggleGroup.Item>
        {/each}
      </ToggleGroup.Root>
    </SettingRow>

    <SettingRow title="Recording visualizer" description="Change how live microphone amplitude is represented. Processing phases keep their own fixed indicators.">
      <ToggleGroup.Root class="grid w-full grid-cols-4" type="single" variant="outline" size="sm" value={settings.overlayVisualizer} onValueChange={chooseVisualizer} aria-label="Recording visualizer" disabled={!settings.overlayEnabled && !previewing}>
        <ToggleGroup.Item value={OverlayVisualizer.OverlayVisualizerBars}>Bars</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayVisualizer.OverlayVisualizerPulse}>Pulse</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayVisualizer.OverlayVisualizerEnvelope}>Envelope</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayVisualizer.OverlayVisualizerMeter}>Thin meter</ToggleGroup.Item>
      </ToggleGroup.Root>
    </SettingRow>

    <SettingRow title="Surface" description="Glass adds layered depth, Solid increases separation, and Minimal removes most decoration. Windows high contrast always takes priority.">
      <ToggleGroup.Root class="grid w-full grid-cols-3" type="single" variant="outline" size="sm" value={settings.overlaySurface} onValueChange={chooseSurface} aria-label="Overlay surface" disabled={!settings.overlayEnabled && !previewing}>
        <ToggleGroup.Item value={OverlaySurface.OverlaySurfaceGlass}>Glass</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlaySurface.OverlaySurfaceSolid}>Solid</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlaySurface.OverlaySurfaceMinimal}>Minimal</ToggleGroup.Item>
      </ToggleGroup.Root>
    </SettingRow>
  </SettingsCard>

  <SettingsCard>
    <SettingRow title="Screen position" description="At the start of each dictation, follow the monitor containing the active application and stay inside its usable work area.">
      <ToggleGroup.Root class="grid w-full grid-cols-3" type="single" variant="outline" size="sm" value={settings.overlayAnchor} onValueChange={chooseAnchor} aria-label="Overlay screen position" disabled={!settings.overlayEnabled && !previewing}>
        {#each anchorChoices as choice (choice.value)}
          <ToggleGroup.Item value={choice.value} class="min-w-0">{choice.label}</ToggleGroup.Item>
        {/each}
      </ToggleGroup.Root>
    </SettingRow>

    <SettingRow title="Edge distance" description="Keep the overlay away from the selected work-area edge. The existing saved position value is preserved.">
      {#snippet control()}<Badge variant="secondary">{settings.overlayTopOffset} px</Badge>{/snippet}
      <Slider.Root id="overlay-edge-offset" type="single" min={0} max={240} step={6} value={settings.overlayTopOffset} onValueChange={(value) => (settings.overlayTopOffset = value)} aria-label="Overlay edge distance" disabled={!settings.overlayEnabled && !previewing} />
    </SettingRow>

    <SettingRow title="Visible phases" description="Recording only is quietest. Active adds transcription and cleanup. All phases also confirms delivery and recoverable outcomes.">
      <ToggleGroup.Root class="grid w-full grid-cols-3" type="single" variant="outline" size="sm" value={settings.overlayVisibility} onValueChange={chooseVisibility} aria-label="Overlay visible phases" disabled={!settings.overlayEnabled && !previewing}>
        <ToggleGroup.Item value={OverlayVisibility.OverlayVisibilityRecording}>Recording</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayVisibility.OverlayVisibilityActive}>Active work</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayVisibility.OverlayVisibilityAll}>All phases</ToggleGroup.Item>
      </ToggleGroup.Root>
    </SettingRow>
  </SettingsCard>

  <SettingsCard>
    <SettingRow title="Motion" description="Follow Windows respects the system animation setting. Reduced removes decorative motion while keeping the functional silence countdown live.">
      <ToggleGroup.Root class="grid w-full grid-cols-2" type="single" variant="outline" size="sm" value={settings.overlayMotion} onValueChange={chooseMotion} aria-label="Overlay motion" disabled={!settings.overlayEnabled && !previewing}>
        <ToggleGroup.Item value={OverlayMotion.OverlayMotionSystem}>Follow Windows</ToggleGroup.Item>
        <ToggleGroup.Item value={OverlayMotion.OverlayMotionReduced}>Reduced</ToggleGroup.Item>
      </ToggleGroup.Root>
    </SettingRow>

    <SettingRow title="Size" description="Scale the complete overlay while preserving its proportions and Windows DPI behavior.">
      {#snippet control()}<Badge variant="secondary">{settings.overlaySizePercent}%</Badge>{/snippet}
      <Slider.Root id="overlay-size" type="single" min={75} max={150} step={5} value={settings.overlaySizePercent} onValueChange={(value) => (settings.overlaySizePercent = value)} aria-label="Overlay size" disabled={!settings.overlayEnabled && !previewing} />
    </SettingRow>

    <SettingRow title="Opacity" description="Adjust the transparency of the complete status surface. High contrast mode always uses full opacity.">
      {#snippet control()}<Badge variant="secondary">{settings.overlayOpacityPercent}%</Badge>{/snippet}
      <Slider.Root id="overlay-opacity" type="single" min={40} max={100} step={5} value={settings.overlayOpacityPercent} onValueChange={(value) => (settings.overlayOpacityPercent = value)} aria-label="Overlay opacity" disabled={!settings.overlayEnabled && !previewing} />
    </SettingRow>

    <SettingRow title="Glow strength" description="Reduce the colored bloom without changing status colors or glyphs. Minimal and high contrast surfaces suppress it.">
      {#snippet control()}<Badge variant="secondary">{settings.overlayGlowPercent}%</Badge>{/snippet}
      <Slider.Root id="overlay-glow" type="single" min={0} max={100} step={5} value={settings.overlayGlowPercent} onValueChange={(value) => (settings.overlayGlowPercent = value)} aria-label="Overlay glow strength" disabled={!settings.overlayEnabled && !previewing} />
    </SettingRow>
  </SettingsCard>
</div>
