<script lang="ts">
  import QuickSaveStatus from "../settings/QuickSaveStatus.svelte";
  import SpeechModelControls from "../settings/SpeechModelControls.svelte";
  import type { QuickSettingsPatch } from "$lib/stores/editor.svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Popover from "$lib/components/ui/popover";
  import { Purpose } from "$bindings/savedconnection";
  import type { Settings } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import { cn } from "$lib/utils";

  let {
    settings,
    editor,
    disabled,
    onAddConnection,
    onOpenSettings,
  }: {
    settings: Settings;
    editor: SettingsEditor;
    disabled: boolean;
    onAddConnection: (purpose: Purpose) => void;
    onOpenSettings: () => void;
  } = $props();
  let open = $state(false);
  const speech = $derived(settings.textToSpeech);
  const busy = $derived(disabled || editor.isQuickSettingsPending("speech-controls"));
  function update(textToSpeech: QuickSettingsPatch["textToSpeech"]) {
    return editor.updateQuickSettings({ textToSpeech }, "speech-controls");
  }
</script>

<div class="flex min-w-0 items-center gap-2" role="group" aria-label="Speech quick settings">
  <Popover.Root bind:open>
    <Popover.Trigger
      {disabled}
      aria-label="Speech settings"
      title="Speech settings"
      class={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-9 gap-2 px-2")}
    >
      <ProviderIcon profile={speech.compatibilityProfile} size={20} />
      <span class="text-[13px]">Speech</span>
      <ChevronDownIcon class="size-3 text-muted-foreground" />
    </Popover.Trigger>
    <Popover.Content role="dialog" aria-label="Speech settings" class="space-y-4">
      <h3 class="text-sm font-semibold">Text to speech</h3>
      <div class="space-y-1.5">
        <label for="home-speech-connection" class="text-xs font-medium">Connection</label>
        <ConnectionSelect
          id="home-speech-connection"
          catalog={settings.savedConnections}
          purpose={Purpose.Speech}
          disabled={busy}
          onAdd={() => {
            open = false;
            onAddConnection(Purpose.Speech);
          }}
          onChange={(change) => editor.changeConnection(change)}
        />
      </div>
      <SpeechModelControls
        {settings}
        compact
        immediate
        busy={busy || !settings.savedConnections.selected?.speech}
        models={editor.ttsConnectionStale || editor.connectionResultStale(Purpose.Speech, settings)
          ? []
          : (editor.ttsConnection?.modelIDs ?? [])}
        modelsBusy={editor.ttsConnectionTesting}
        voices={editor.voicesFor(settings)}
        voicesBusy={editor.voicesBusy}
        onChooseModel={(model) => update({ model })}
        onVoice={(voice) => update({ voice })}
        onSpeed={(speed) => update({ speed })}
        onOptions={(options) => update({ options })}
        onDiscoverModels={() => editor.testTextToSpeechConnection(settings, "")}
        onDiscoverVoices={() => editor.discoverVoices(true)}
      />
      <QuickSaveStatus
        fields={["speech-controls"]}
        pending={editor.quickSettingsPending}
        saved={editor.quickSettingsSaved}
        failed={editor.quickSettingsFailed}
      />
      <Button
        variant="outline"
        size="sm"
        class="w-full"
        onclick={() => {
          open = false;
          onOpenSettings();
        }}
      >
        All speech settings
      </Button>
    </Popover.Content>
  </Popover.Root>
  <span
    class="hidden min-w-0 truncate text-xs text-muted-foreground @min-[540px]:inline"
    title={speech.voice || undefined}
  >
    {speech.voice || "Choose a voice"}
  </span>
</div>
