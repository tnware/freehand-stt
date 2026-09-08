<script lang="ts">
  import VoiceTranscriptionSettings from "./VoiceTranscriptionSettings.svelte";
  import MicIcon from "@lucide/svelte/icons/mic";
  import TextCursorInputIcon from "@lucide/svelte/icons/text-cursor-input";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import * as Popover from "$lib/components/ui/popover";
  import { buttonVariants } from "$lib/components/ui/button";
  import QuickControls from "./QuickControls.svelte";
  import QuickSettings from "./QuickSettings.svelte";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import type { Settings } from "$lib/state";
  import { cn } from "$lib/utils";

  let {
    editor,
    settings,
    showCapture,
    disabled,
    onAddConnection,
    onOpenServerSettings,
    onOpenProcessingSettings,
    onOpenAudioSettings,
    onOpenGeneralSettings,
  }: {
    editor: SettingsEditor;
    settings: Settings;
    showCapture: boolean;
    disabled: boolean;
    onAddConnection: (purpose: Purpose) => void;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    onOpenAudioSettings: () => void;
    onOpenGeneralSettings: () => void;
  } = $props();
  type Panel = "audio" | "stt" | "cleanup" | "delivery";
  let activePanel = $state<Panel | null>(null);
  const panels = $derived<Panel[]>(
    showCapture ? ["audio", "stt", "cleanup", "delivery"] : ["stt", "cleanup"],
  );
  const labels = {
    audio: "Audio settings",
    stt: "Transcription settings",
    cleanup: "Cleanup settings",
    delivery: "Delivery settings",
  };
  function openSettings(open: () => void) {
    activePanel = null;
    open();
  }
</script>

<div class="flex h-12 min-w-0 items-center gap-1" role="group" aria-label="Quick settings">
  {#each panels as panel (panel)}
    <Popover.Root
      open={!disabled && activePanel === panel}
      onOpenChange={(open) => (activePanel = open ? panel : null)}
    >
      <Popover.Trigger
        {disabled}
        aria-label={labels[panel]}
        title={labels[panel]}
        class={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-9 min-w-0 gap-2 px-2")}
      >
        {#if panel === "audio"}<MicIcon class="size-4" />
        {:else if panel === "delivery"}<TextCursorInputIcon class="size-4" />
        {:else}<ProviderIcon
            profile={panel === "stt"
              ? showCapture
                ? settings.voiceTranscription.compatibilityProfile
                : settings.compatibilityProfile
              : settings.postProcessing.compatibilityProfile}
            size={20}
          />
          <span class="hidden text-[13px] @min-[540px]:inline"
            >{panel === "stt" ? "Transcription" : "Cleanup"}</span
          >
          {#if panel === "cleanup"}<span
              class={settings.postProcessing.enabled
                ? "size-1.5 rounded-full bg-success"
                : "size-1.5 rounded-full bg-muted-foreground"}
              aria-label={settings.postProcessing.enabled ? "Enabled" : "Disabled"}
            ></span>{/if}
        {/if}
        <ChevronDownIcon class="size-3 text-muted-foreground" />
      </Popover.Trigger>
      <Popover.Content role="dialog" aria-label={labels[panel]}>
        {#if panel === "stt" && showCapture}
          <VoiceTranscriptionSettings
            {editor}
            {settings}
            {disabled}
            onAddConnection={(purpose) => {
              activePanel = null;
              onAddConnection(purpose);
            }}
          />
        {:else if panel === "audio" || panel === "delivery"}
          <QuickControls
            {settings}
            devices={editor.devices}
            pending={editor.quickSettingsPending}
            savedField={editor.quickSettingsSaved}
            {disabled}
            section={panel}
            onUpdate={(patch, field) => editor.updateQuickSettings(patch, field)}
            onOpenAudioSettings={() => openSettings(onOpenAudioSettings)}
            onOpenDeliverySettings={() => openSettings(onOpenGeneralSettings)}
          />
        {:else}
          <QuickSettings
            embedded
            showCapture={false}
            showTranscription={panel === "stt"}
            showCleanup={panel === "cleanup"}
            {settings}
            devices={editor.devices}
            processingProfiles={editor.processingProfiles}
            connection={editor.connection}
            processingConnection={editor.processingConnection}
            sttStale={editor.sttConnectionStale ||
              editor.connectionResultStale(Purpose.Transcription, settings)}
            processingStale={editor.processingConnectionStale ||
              editor.connectionResultStale(Purpose.Cleanup, settings)}
            pending={editor.quickSettingsPending}
            savedField={editor.quickSettingsSaved}
            failedField={editor.quickSettingsFailed}
            sttTesting={editor.sttConnectionTesting}
            processingTesting={editor.processingConnectionTesting}
            onAddConnection={(purpose) => {
              activePanel = null;
              onAddConnection(purpose);
            }}
            onChangeConnection={(change) => editor.changeConnection(change)}
            onUpdate={(patch, field) => editor.updateQuickSettings(patch, field)}
            onTestConnection={() => editor.testConnection(editor.applied, "")}
            onTestProcessingConnection={() =>
              editor.testPostProcessingConnection(editor.applied, "")}
            disabled={disabled || editor.saving}
            onOpenServerSettings={() => openSettings(onOpenServerSettings)}
            onOpenProcessingSettings={() => openSettings(onOpenProcessingSettings)}
            onOpenAudioSettings={() => openSettings(onOpenAudioSettings)}
            onOpenDeliverySettings={() => openSettings(onOpenGeneralSettings)}
          />
        {/if}
        {#if panel === "audio" || panel === "delivery"}<p
            class="mt-3 border-t border-hairline pt-3 text-xs text-muted-foreground"
          >
            Changes apply immediately.
          </p>{/if}
      </Popover.Content>
    </Popover.Root>
  {/each}
  <span class="sr-only" role="status" aria-live="polite"
    >{editor.quickSettingsPending.length
      ? "Saving…"
      : editor.quickSettingsSaved
        ? "Saved"
        : ""}</span
  >
</div>
