<script lang="ts">
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";

  let {
    label,
    disabled = false,
    onclick,
  }: {
    label: string;
    disabled?: boolean;
    onclick: () => void;
  } = $props();
</script>

<TooltipButton
  {label}
  {disabled}
  {onclick}
  size="sm"
  variant="ghost"
  class="shrink-0 px-2"
>
  <SettingsIcon class="size-3.5" />
  <span class="settings-label">Settings</span>
</TooltipButton>

<style>
  .settings-label {
    display: none;
  }
  @container (min-width: 540px) {
    .settings-label {
      display: inline;
    }
  }
</style>
