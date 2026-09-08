<script lang="ts">
  import * as Slider from "$lib/components/ui/slider";
  import { Badge } from "$lib/components/ui/badge";
  let {
    id,
    value,
    supported,
    disabled = false,
    compact = false,
    onChange,
  }: {
    id: string;
    value: number;
    supported: boolean;
    disabled?: boolean;
    compact?: boolean;
    onChange: (speed: number) => boolean | Promise<boolean>;
  } = $props();
  let pending = $state(false);
  let preview = $state<number | null>(null);
  const shown = $derived(preview ?? value);
  async function commit(speed: number) {
    pending = true;
    try {
      await onChange(speed);
    } finally {
      preview = null;
      pending = false;
    }
  }
</script>

<div class={compact ? "space-y-2" : "space-y-2 px-5 py-4"}>
  <label for={id} class="text-sm font-medium">Speaking speed</label>
  <div class="flex items-center gap-3">
    <Slider.Root
      {id}
      type="single"
      min={0.25}
      max={4}
      step={0.05}
      value={shown}
      disabled={disabled || pending || !supported}
      onValueChange={(speed) => (preview = speed)}
      onValueCommit={commit}
      aria-label="Speech playback speed"
      aria-describedby={`${id}-help`}
    />
    <Badge variant="outline" class="min-w-14 justify-center font-mono">{shown.toFixed(2)}×</Badge>
  </div>
  <p
    id={`${id}-help`}
    class={supported ? "sr-only" : "text-xs leading-relaxed text-muted-foreground"}
  >
    {supported
      ? "Applies to the next generation, from 0.25× to 4×."
      : "This model profile does not support adjustable speed."}
  </p>
</div>
