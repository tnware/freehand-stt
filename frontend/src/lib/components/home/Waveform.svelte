<script lang="ts">
  let {
    active = false,
    quiet = false,
    held = false,
    history,
  }: {
    active?: boolean;
    /** True when the native VAD has stabilized on silence. */
    quiet?: boolean;
    /** The take is captured and in flight: it is history now, not a level. */
    held?: boolean;
    /** Capture amplitude, oldest first. Each entry is one reading. */
    history: number[];
  } = $props();

  // Floored just above zero so silence reads as a quiet meter, not a broken
  // one. No transition: the bars scroll, so consecutive values belong to
  // different moments and animating between them is meaningless.
  const height = (level: number) => (0.04 + 0.96 * level) * 100;
  const fill = $derived(
    held ? "bg-meter-rest/60" : !active ? "bg-meter-rest" : quiet ? "bg-meter-rest" : "bg-meter",
  );
</script>

<!-- The transport owns the meter's height so a narrow window can shorten it
     without the bars losing their floor. -->
<div class="meter flex min-w-0 flex-1 items-end gap-[2px]" aria-hidden="true">
  {#each history as level, index (index)}
    <span
      class="min-h-[2px] w-full min-w-0 flex-1 rounded-[1px] {fill}"
      style="height: {height(level)}%"
    ></span>
  {/each}
</div>

<style>
  .meter {
    height: var(--meter-height, 3.875rem);
  }
</style>
