<script lang="ts">
  import type { NeMoOptions } from "$bindings/compatibility";
  import { Switch } from "$lib/components/ui/switch";
  import { Input } from "$lib/components/ui/input";
  import SettingsDisclosure from "./SettingsDisclosure.svelte";

  let {
    options,
    realtime = false,
    managed = false,
    disabled = false,
    onChange,
  }: {
    options: NeMoOptions;
    realtime?: boolean;
    managed?: boolean;
    disabled?: boolean;
    onChange: (options: NeMoOptions) => void;
  } = $props();
  const uid = $props.id();
  function update(changes: Partial<NeMoOptions>) {
    onChange({ ...options, ...changes });
  }
</script>

<SettingsDisclosure
  title="NeMo transcription controls"
  description="Punctuation, text formatting, and server processing"
>
  <div class="space-y-4 py-3">
    <div class="flex items-center justify-between gap-3">
      <label for={`${uid}-punctuation`} class="text-[13px] font-medium"
        >Automatic punctuation</label
      >
      <Switch
        id={`${uid}-punctuation`}
        checked={!options.disablePunctuation}
        {disabled}
        onCheckedChange={(enabled) => update({ disablePunctuation: !enabled })}
      />
    </div>
    <div class="flex items-center justify-between gap-3">
      <label for={`${uid}-normalize`} class="text-[13px] font-medium"
        >Normalize numbers and dates</label
      >
      <Switch
        id={`${uid}-normalize`}
        checked={options.normalize}
        {disabled}
        onCheckedChange={(normalize) => update({ normalize })}
      />
    </div>
    <p class="text-xs text-muted-foreground">
      Requests inverse text normalization, such as “twenty one” → “21”. Requires
      language-specific grammars configured on the server. Off preserves spoken
      wording.
    </p>
    <div class="flex items-center justify-between gap-3">
      <label for={`${uid}-profanity`} class="text-[13px] font-medium"
        >Profanity filter</label
      >
      <Switch
        id={`${uid}-profanity`}
        checked={options.profanityFilter}
        {disabled}
        onCheckedChange={(profanityFilter) => update({ profanityFilter })}
      />
    </div>
    <p class="text-xs text-muted-foreground">
      Requires a word list configured on the server. Freehand cannot verify
      whether the list or normalization grammars are loaded.
    </p>
    {#if managed}<p class="text-xs text-muted-foreground">
        Managed runtimes do not install normalization grammars or a profanity
        word list. These requests have no effect without those assets.
      </p>{/if}
    {#if realtime}
      <div class="space-y-2">
        <label for={`${uid}-endpointing`} class="text-[13px] font-medium"
          >Server endpointing delay (ms)</label
        >
        <Input
          id={`${uid}-endpointing`}
          type="number"
          min={0}
          max={10000}
          step={100}
          value={options.endpointingMilliseconds}
          {disabled}
          aria-invalid={options.endpointingMilliseconds !== 0 &&
            (options.endpointingMilliseconds < 100 ||
              options.endpointingMilliseconds > 10000)}
          onchange={(event) => {
            const value = event.currentTarget.valueAsNumber;
            update({
              endpointingMilliseconds: Number.isFinite(value) ? value : -1,
            });
          }}
        />
        <p class="text-xs text-muted-foreground">
          0 uses the server default. Use 100–10,000 ms to adjust when a pause
          finishes an utterance. Endpointing must already be enabled on the
          server; this does not change Freehand’s microphone stop controls.
        </p>
        {#if managed}<p class="text-xs text-muted-foreground">
            Managed NeMo uses its default endpointing configuration. This
            override cannot enable endpointing.
          </p>{/if}
      </div>
    {/if}
  </div>
</SettingsDisclosure>
