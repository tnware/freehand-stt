<script lang="ts">
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
          {disabled}
          onAdd={() => {
            open = false;
            onAddConnection(Purpose.Speech);
          }}
          onChange={(change) => editor.changeConnection(change)}
        />
      </div>
      <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-xs">
        <dt class="text-muted-foreground">Model</dt>
        <dd class="break-words text-right">{speech.model || "Not selected"}</dd>
        <dt class="text-muted-foreground">Voice</dt>
        <dd class="break-words text-right">{speech.voice || "Not selected"}</dd>
      </dl>
      <Button
        variant="outline"
        size="sm"
        class="w-full"
        onclick={() => {
          open = false;
          onOpenSettings();
        }}
      >
        Model and voice settings
      </Button>
      <p class="border-t border-hairline pt-3 text-xs text-muted-foreground">
        Connection changes apply immediately. Your draft stays in the composer.
      </p>
    </Popover.Content>
  </Popover.Root>
  <span
    class="hidden min-w-0 truncate text-xs text-muted-foreground @min-[540px]:inline"
    title={speech.voice || undefined}
  >
    {speech.voice || "Choose a voice"}
  </span>
</div>
