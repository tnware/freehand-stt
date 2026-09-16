<script lang="ts">
  import VoicePicker from "$lib/components/settings/VoicePicker.svelte";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import { VoiceScope, type VoicesResult } from "$bindings/inference";

  const sidebar = new URLSearchParams(location.search).has("sidebar");
  let voiceBusy = $state(false);
  let modelBusy = $state(false);
  let voice = $state("default");
  let result = $state<VoicesResult | null>(null);
  let metadataStatus = $state<"idle" | "failed">("idle");
  function fail() {
    voiceBusy = modelBusy = false;
    metadataStatus = "failed";
    result = {
      voices: [],
      scope: VoiceScope.VoiceScopeModel,
      errorKind: "timeout",
      httpStatus: 0,
      latencyMilliseconds: 0,
      truncated: false,
    };
  }
</script>

<main class="h-screen overflow-auto bg-background p-4 text-foreground">
  <section
    aria-label="Picker controls"
    class="space-y-4"
    style:width={sidebar ? "220px" : "360px"}
  >
    <RuntimeModelPicker
      id="fixture-model"
      value="fixture/model"
      compact
      {sidebar}
      busy={modelBusy}
      {metadataStatus}
      onChoose={() => true}
      onDiscover={() => (modelBusy = true)}
    />
    <VoicePicker
      id="fixture-voice"
      bind:value={voice}
      compact
      {sidebar}
      supported
      allowedVoices={["default", "Aria"]}
      busy={voiceBusy}
      {result}
      onDiscover={() => (voiceBusy = true)}
    />
  </section>
  <button class="mt-8" onclick={fail}>Fail refreshes</button>
</main>
