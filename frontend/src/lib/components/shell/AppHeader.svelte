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

<header class="app-header flex h-[72px] shrink-0 items-center gap-7 px-5">
  <div class="flex shrink-0 items-center gap-2.5">
    <span class="grid size-9 place-items-center rounded-xl bg-accent-wash text-accent-text"
      ><BrandMark /></span
    >
    <h1 class="font-display text-[17px] font-semibold tracking-tight">Freehand</h1>
  </div>

  <!-- One mode selector owns navigation; the selected surface and accent
       remain separate from the keyboard focus indicator. -->
  <Tabs.Root bind:value={inputMode} class="min-w-0">
    <Tabs.List
      variant="default"
      class="mode-tabs h-10 gap-1 rounded-xl border border-hairline bg-well p-1"
      aria-label="Input source"
    >
      <Tabs.Trigger
        value="voice"
        disabled={fileWorking}
        class="mode-tab h-8 rounded-lg px-3 text-[13px] after:hidden data-active:bg-card data-active:text-accent-text dark:data-active:border-transparent dark:data-active:bg-card dark:data-active:text-accent-text"
      >
        <MicIcon data-icon="inline-start" />
        Voice
      </Tabs.Trigger>
      <Tabs.Trigger
        value="file"
        disabled={voiceActive}
        class="mode-tab h-8 rounded-lg px-3 text-[13px] after:hidden data-active:bg-card data-active:text-accent-text dark:data-active:border-transparent dark:data-active:bg-card dark:data-active:text-accent-text"
      >
        <FileAudioIcon data-icon="inline-start" />
        Audio file
      </Tabs.Trigger>
      <Tabs.Trigger
        value="tts"
        disabled={voiceActive || fileWorking}
        class="mode-tab h-8 rounded-lg px-3 text-[13px] after:hidden data-active:bg-card data-active:text-accent-text dark:data-active:border-transparent dark:data-active:bg-card dark:data-active:text-accent-text"
      >
        <Volume2Icon data-icon="inline-start" />
        Text to speech
      </Tabs.Trigger>
    </Tabs.List>
  </Tabs.Root>

  <div class="ml-auto flex shrink-0 items-center gap-2">
    {#if toggleShortcut}
      <div class="hidden items-center gap-2 px-2 py-1 min-[900px]:flex">
        <MicIcon class="size-3 text-muted-foreground" aria-hidden="true" />
        <ShortcutKeys value={toggleShortcut} label="Recording shortcut" />
      </div>
    {/if}
    <Button
      variant="ghost"
      size="icon-sm"
      onclick={onSettings}
      aria-label={settingsOpen ? "Focus Settings" : "Open Settings"}
      title={settingsOpen ? "Focus Settings" : "Open Settings"}
    >
      <SettingsIcon />
    </Button>
  </div>
</header>

<style>
  .app-header :global(.mode-tab[data-state="active"]) {
    background: var(--card);
    color: var(--accent-text);
    box-shadow: 0 1px 4px rgb(0 0 0 / 8%);
  }
  @media (max-width: 699px) {
    .app-header {
      height: 64px;
      gap: 1rem;
      padding-inline: 0.75rem;
    }
    .app-header h1 {
      display: none;
    }
    .app-header :global(.mode-tab) {
      padding-inline: 0.625rem;
    }
  }
  @media (forced-colors: active) {
    .app-header :global(.mode-tab[data-state="active"]) {
      outline: 2px solid Highlight;
      outline-offset: -2px;
    }
  }
</style>
