<script lang="ts">
  import { getContext } from "svelte";
  import { SETTINGS_NAVIGATION, type SettingsSectionID } from "$lib/navigation";
  import { Service as WindowingService } from "$bindings/windowing";
  import type { Settings } from "$lib/state";
  import BookOpenIcon from "@lucide/svelte/icons/book-open";
  import { Button } from "$lib/components/ui/button";
  let { settings, voice = false }: { settings: Settings; voice?: boolean } = $props();
  const navigate = getContext<((id: SettingsSectionID) => void) | undefined>(SETTINGS_NAVIGATION);
  let error = $state("");
  const enabled = $derived(voice ? settings.vocabulary.voice : settings.vocabulary.files);
  async function open() {
    if (navigate) {
      navigate("vocabulary");
      return;
    }
    try {
      await WindowingService.OpenSettings("vocabulary");
    } catch {
      error = "Could not open Vocabulary settings.";
    }
  }
</script>

<div class="flex flex-wrap items-center justify-between gap-2 border-t border-hairline pt-3">
  <div>
    <p class="text-xs font-medium">Shared vocabulary</p>
    <p class="mt-1 text-xs text-muted-foreground">
      {enabled ? "On for supported models" : "Off for this workflow"}
    </p>
  </div>
  <Button variant="ghost" size="sm" onclick={open}><BookOpenIcon class="size-4" />Vocabulary</Button
  >
  {#if error}<p class="w-full text-xs text-destructive" role="alert">{error}</p>{/if}
</div>
