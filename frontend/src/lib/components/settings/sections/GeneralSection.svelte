<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import { Label } from "$lib/components/ui/label";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { AppearanceMode, InsertionMode, type Settings } from "$lib/state";

  let { settings = $bindable() }: { settings: Settings } = $props();

  let deliveryMode = $derived(
    settings.autoInsert ? InsertionMode.DirectInput : InsertionMode.ManualCopy,
  );
  let appearanceRestartRequired = $derived(
    !settings.useMica &&
      settings.useMica === settings.micaActive &&
      settings.appearanceMode !== settings.appearanceModeActive,
  );

  const appearanceModes = [
    {
      value: AppearanceMode.AppearanceModeSystem,
      label: "System",
      description: "Follow Windows",
    },
    {
      value: AppearanceMode.AppearanceModeLight,
      label: "Light",
      description: "Always light",
    },
    {
      value: AppearanceMode.AppearanceModeDark,
      label: "Dark",
      description: "Always dark",
    },
  ];

  const chooseAppearanceMode = (value: string) => {
    const selected = appearanceModes.find((mode) => mode.value === value);
    if (selected && !settings.useMica) settings.appearanceMode = selected.value;
  };

  const chooseDeliveryMode = (value: string) => {
    if (value === InsertionMode.DirectInput) settings.autoInsert = true;
    if (value === InsertionMode.ManualCopy) settings.autoInsert = false;
  };
</script>

<SettingsCard>
  <SettingRow
    title="Start with Windows"
    description="Launch quietly in the tray when you sign in."
  >
    {#snippet control()}
      <Switch
        id="start-with-windows"
        bind:checked={settings.startWithWindows}
        aria-label="Start with Windows"
      />
    {/snippet}
  </SettingRow>

  <SettingRow
    title="Show window when launched"
    description="Open this window on a normal manual launch. Windows sign-in launches always remain tray-only."
  >
    {#snippet control()}
      <Switch
        id="show-window-on-launch"
        bind:checked={settings.showWindowOnLaunch}
        aria-label="Show window when launched"
      />
    {/snippet}
  </SettingRow>

  <SettingRow
    title="Check for updates automatically"
    description="Check GitHub Releases in the background. Freehand never applies an update without asking you."
  >
    {#snippet control()}
      <Switch
        id="check-for-updates"
        bind:checked={settings.checkForUpdates}
        aria-label="Check for updates automatically"
      />
    {/snippet}
  </SettingRow>

</SettingsCard>

<SettingsCard>
  <SettingRow
    title="Color mode"
    description={settings.useMica
      ? "Mica follows the Windows light or dark setting. Your solid-window preference is preserved for when Mica is off."
      : "Follow Windows or keep Freehand independently light or dark. Applies after restarting the app."}
  >
    {#snippet control()}
      {#if appearanceRestartRequired}
        <Badge variant="secondary">Restart required</Badge>
      {/if}
    {/snippet}

    <div class:opacity-60={settings.useMica}>
      <RadioGroup.Root
        class="grid-cols-3 gap-2"
        value={settings.appearanceMode}
        onValueChange={chooseAppearanceMode}
        disabled={settings.useMica}
        aria-label="Color mode"
      >
        {#each appearanceModes as mode (mode.value)}
          <Label
            for={`appearance-${mode.value}`}
            class={settings.useMica
              ? "flex cursor-not-allowed items-center gap-2.5 rounded-lg border border-hairline bg-background/35 px-3 py-2.5 has-data-checked:border-primary/30 has-data-checked:bg-primary/5"
              : "flex cursor-pointer items-center gap-2.5 rounded-lg border border-hairline bg-background/35 px-3 py-2.5 transition-colors has-data-checked:border-primary/30 has-data-checked:bg-primary/5 hover:bg-accent/55"}
          >
            <RadioGroup.Item id={`appearance-${mode.value}`} value={mode.value} />
            <span class="min-w-0">
              <span class="block text-xs font-medium text-foreground">{mode.label}</span>
              <span class="block text-[10.5px] leading-relaxed text-muted-foreground">
                {mode.description}
              </span>
            </span>
          </Label>
        {/each}
      </RadioGroup.Root>
    </div>
  </SettingRow>

  <SettingRow
    title="Use Windows Mica backdrop"
    description="Show the Windows system material through the app shell where supported. Mica always follows the Windows light or dark setting and applies after restarting the app."
  >
    {#snippet control()}
      <div class="flex items-center gap-3">
        {#if settings.useMica !== settings.micaActive}
          <Badge variant="secondary">Restart required</Badge>
        {/if}
        <Switch
          id="use-mica"
          bind:checked={settings.useMica}
          aria-label="Use Windows Mica backdrop"
        />
      </div>
    {/snippet}
  </SettingRow>
</SettingsCard>

<SettingsCard>
  <SettingRow
    title="Transcript delivery"
    description="Choose what Freehand does with a completed microphone transcript. Focus safety always applies."
  >
    <RadioGroup.Root
      class="grid-cols-1 gap-2"
      orientation="vertical"
      value={deliveryMode}
      onValueChange={chooseDeliveryMode}
      aria-label="Transcript delivery mode"
    >
      <Label
        for="delivery-direct-input"
        class="flex cursor-pointer items-start gap-3 rounded-lg border border-hairline bg-background/35 px-3 py-3 transition-colors hover:bg-accent/55"
      >
        <RadioGroup.Item
          id="delivery-direct-input"
          value={InsertionMode.DirectInput}
          class="mt-0.5"
        />
        <span class="min-w-0">
          <span class="block text-xs font-medium text-foreground">Direct input</span>
          <span class="mt-0.5 block text-[11px] leading-relaxed text-muted-foreground">
            Type Unicode directly into the application that was focused when recording started. This is the default and does not touch the clipboard.
          </span>
        </span>
      </Label>

      <Label
        for="delivery-manual-copy"
        class="flex cursor-pointer items-start gap-3 rounded-lg border border-hairline bg-background/35 px-3 py-3 transition-colors hover:bg-accent/55"
      >
        <RadioGroup.Item
          id="delivery-manual-copy"
          value={InsertionMode.ManualCopy}
          class="mt-0.5"
        />
        <span class="min-w-0">
          <span class="block text-xs font-medium text-foreground">Manual copy</span>
          <span class="mt-0.5 block text-[11px] leading-relaxed text-muted-foreground">
            Keep every completed transcript in Freehand until you explicitly choose Copy transcript. Nothing is inserted or copied automatically.
          </span>
        </span>
      </Label>

      <Label
        for="delivery-clipboard-paste"
        aria-disabled="true"
        class="flex cursor-not-allowed items-start gap-3 rounded-lg border border-hairline bg-muted/25 px-3 py-3 opacity-65"
      >
        <RadioGroup.Item
          id="delivery-clipboard-paste"
          value={InsertionMode.ClipboardPaste}
          class="mt-0.5"
          disabled
        />
        <span class="min-w-0 flex-1">
          <span class="flex flex-wrap items-center gap-2 text-xs font-medium text-foreground">
            Clipboard paste
            <Badge variant="secondary">Deferred</Badge>
          </span>
          <span class="mt-0.5 block text-[11px] leading-relaxed text-muted-foreground">
            A future compatibility mode for applications that reject direct input. It remains unavailable until complete clipboard preservation and conditional restoration are implemented.
          </span>
        </span>
      </Label>
    </RadioGroup.Root>
  </SettingRow>
</SettingsCard>
