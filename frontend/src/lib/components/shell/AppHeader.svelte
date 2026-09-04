<script lang="ts">
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import MicIcon from "@lucide/svelte/icons/mic";
  import SettingsIcon from "@lucide/svelte/icons/sliders-horizontal";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import { Button } from "$lib/components/ui/button";
  import * as Tabs from "$lib/components/ui/tabs";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import type { Settings } from "$lib/state";

  let {
    inputMode = $bindable("voice"),
    settings,
    voiceActive = false,
    fileWorking = false,
    onSettings,
    settingsOpen = false,
  }: {
    inputMode: string;
    settings: Settings | null;
    voiceActive?: boolean;
    fileWorking?: boolean;
    onSettings: () => void;
    settingsOpen?: boolean;
  } = $props();

  // The global shortcut is the product's primary interaction, so it stays on
  // screen rather than appearing only while idle. Endpoint health moved out of
  // this row entirely: at 10px in a corner it was unreadable, and the rack
  // states it per stage where the endpoint is actually configured.
  const toggleShortcut = $derived(settings?.toggleShortcut ?? "");
</script>

<header
  class="flex h-12 shrink-0 items-center gap-6 overflow-hidden border-b border-hairline bg-layer-fill px-4"
>
  <div class="flex shrink-0 items-center gap-2.5">
    <BrandMark />
    <h1 class="truncate text-sm font-semibold">Freehand</h1>
  </div>

  <!--
    Mode is a line of text with a rule under the active one. A segmented pill
    would be a second, competing container on a header that already has the
    wordmark and the shortcut in it.
  -->
  <Tabs.Root bind:value={inputMode} class="h-full">
    <Tabs.List
      variant="line"
      class="h-full gap-5 p-0 group-data-horizontal/tabs:h-full"
      aria-label="Input source"
    >
      <Tabs.Trigger
        value="voice"
        disabled={fileWorking}
        class="h-full rounded-none px-0 text-[12.5px] after:bottom-px after:bg-primary"
      >
        <MicIcon data-icon="inline-start" />
        Voice
      </Tabs.Trigger>
      <Tabs.Trigger
        value="file"
        disabled={voiceActive}
        class="h-full rounded-none px-0 text-[12.5px] after:bottom-px after:bg-primary"
      >
        <FileAudioIcon data-icon="inline-start" />
        Audio file
      </Tabs.Trigger>
      <Tabs.Trigger
        value="tts"
        disabled={voiceActive || fileWorking}
        class="h-full rounded-none px-0 text-[12.5px] after:bottom-px after:bg-primary"
      >
        <Volume2Icon data-icon="inline-start" />
        Text to speech
      </Tabs.Trigger>
    </Tabs.List>
  </Tabs.Root>

  <div class="ml-auto flex shrink-0 items-center gap-2">
    {#if toggleShortcut}
      <div class="hidden items-center gap-2 rounded-lg border border-border bg-control-fill px-2 py-1 sm:flex">
        <MicIcon class="size-3 text-muted-foreground" aria-hidden="true" />
        <ShortcutKeys value={toggleShortcut} label="Recording shortcut" />
      </div>
    {/if}
    <Button
      variant="outline"
      size="icon-sm"
      onclick={onSettings}
      aria-label={settingsOpen ? "Focus Settings" : "Open Settings"}
      title={settingsOpen ? "Focus Settings" : "Open Settings"}
    >
      <SettingsIcon />
    </Button>
  </div>
</header>
