<script lang="ts">
  import NativePermissionsCard from "$lib/components/settings/NativePermissionsCard.svelte";
  import { platformPresentation } from "$lib/platform";
  import { Badge } from "$lib/components/ui/badge";
  import { Label } from "$lib/components/ui/label";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { AppearanceMode, InsertionMode, type Settings } from "$lib/state";

  let { settings = $bindable() }: { settings: Settings } = $props();

  const native = $derived(platformPresentation(settings.platform));
  const mica = $derived(native.supportsMica && settings.useMica);

  let deliveryMode = $derived(
    settings.autoInsert ? InsertionMode.DirectInput : InsertionMode.ManualCopy,
  );
  let appearanceRestartRequired = $derived(
    !mica &&
      (!native.supportsMica || settings.useMica === settings.micaActive) &&
      settings.appearanceMode !== settings.appearanceModeActive,
  );

  const appearanceModes = $derived([
    {
      value: AppearanceMode.AppearanceModeSystem,
      label: "System",
      description: native.followSystem,
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
  ]);

  const chooseAppearanceMode = (value: string) => {
    const selected = appearanceModes.find((mode) => mode.value === value);
    if (selected && !mica) settings.appearanceMode = selected.value;
  };

  const chooseDeliveryMode = (value: string) => {
    if (value === InsertionMode.DirectInput) settings.autoInsert = true;
    if (value === InsertionMode.ManualCopy) settings.autoInsert = false;
  };
</script>

{#if native.mac}<NativePermissionsCard />{/if}

<SettingsCard>
  <SettingRow
    controlID="start-with-windows"
    title={native.startTitle}
    description={native.startDescription}
  >
    {#snippet control()}
      <Switch
        id="start-with-windows"
        bind:checked={settings.startWithWindows}
        aria-label={native.startTitle}
      />
    {/snippet}
  </SettingRow>

  <SettingRow
    controlID="show-window-on-launch"
    title="Show window when launched"
    description={native.launchDescription}
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
    controlID="check-for-updates"
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
    description={mica
      ? "Mica follows the Windows light or dark setting. Your solid-window preference is preserved for when Mica is off."
      : `${native.followSystem} or keep Freehand independently light or dark. Applies after restarting the app.`}
  >
    {#snippet control()}
      {#if appearanceRestartRequired}
        <Badge variant="secondary">Restart required</Badge>
      {/if}
    {/snippet}

    <div class="@container/appearance" class:opacity-60={mica}>
      <RadioGroup.Root
        class="grid-cols-1 gap-1 @min-[360px]/appearance:grid-cols-3"
        value={settings.appearanceMode}
        onValueChange={chooseAppearanceMode}
        disabled={mica}
        aria-label="Color mode"
      >
        {#each appearanceModes as mode (mode.value)}
          <Label
            for={`appearance-${mode.value}`}
            class={mica
              ? "flex cursor-not-allowed items-center gap-2.5 bg-transparent px-2 py-2 has-data-checked:bg-accent-wash"
              : "flex cursor-pointer items-center gap-2.5 bg-transparent px-2 py-2 transition-colors has-data-checked:bg-accent-wash hover:bg-accent/55"}
          >
            <RadioGroup.Item
              id={`appearance-${mode.value}`}
              value={mode.value}
            />
            <span class="min-w-0">
              <span class="block text-[13px] font-medium text-foreground"
                >{mode.label}</span
              >
              <span class="block text-xs leading-5 text-muted-foreground">
                {mode.description}
              </span>
            </span>
          </Label>
        {/each}
      </RadioGroup.Root>
    </div>
  </SettingRow>

  {#if native.supportsMica}
    <SettingRow
      controlID="use-mica"
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
  {/if}
</SettingsCard>

<SettingsCard>
  <SettingRow
    title="Transcript delivery"
    description="Choose what Freehand does with a completed microphone transcript. Focus safety always applies."
  >
    <RadioGroup.Root
      class="grid-cols-1 gap-1"
      orientation="vertical"
      value={deliveryMode}
      onValueChange={chooseDeliveryMode}
      aria-label="Transcript delivery mode"
    >
      <Label
        for="delivery-direct-input"
        class="flex cursor-pointer items-start gap-3 bg-transparent px-2 py-2 transition-colors has-data-checked:bg-accent-wash hover:bg-accent/55"
      >
        <RadioGroup.Item
          id="delivery-direct-input"
          value={InsertionMode.DirectInput}
          class="mt-0.5"
        />
        <span class="min-w-0">
          <span class="block text-[13px] font-medium text-foreground"
            >Direct input</span
          >
          <span class="mt-0.5 block text-xs leading-5 text-muted-foreground">
            Type Unicode directly into the application that was focused when
            recording started. This is the default and does not touch the
            clipboard.
          </span>
        </span>
      </Label>

      <Label
        for="delivery-manual-copy"
        class="flex cursor-pointer items-start gap-3 bg-transparent px-2 py-2 transition-colors has-data-checked:bg-accent-wash hover:bg-accent/55"
      >
        <RadioGroup.Item
          id="delivery-manual-copy"
          value={InsertionMode.ManualCopy}
          class="mt-0.5"
        />
        <span class="min-w-0">
          <span class="block text-[13px] font-medium text-foreground"
            >Manual copy</span
          >
          <span class="mt-0.5 block text-xs leading-5 text-muted-foreground">
            Keep every completed transcript in Freehand until you explicitly
            choose Copy transcript. Nothing is inserted or copied automatically.
          </span>
        </span>
      </Label>
    </RadioGroup.Root>
  </SettingRow>
</SettingsCard>
