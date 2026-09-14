<script lang="ts">
  let {
    label,
    tone = "idle",
  }: {
    /** What travels between the two stages: "audio" or "text". */
    label: string;
    tone?: "idle" | "live" | "blocked";
  } = $props();
</script>

<!-- Naming the payload is the point: it is what makes the chain explain that
     stage 02 turns audio into text, rather than just sequencing four cards. -->
<div
  class="connector flex w-10 shrink-0 flex-col items-center justify-center gap-1.5"
  data-tone={tone}
  aria-hidden="true"
>
  <span class="font-mono text-[9px] tracking-[0.04em] text-ink-quiet"
    >{label}</span
  >
  <svg viewBox="0 0 40 8" width="40" height="8" fill="none" aria-hidden="true">
    {#if tone === "blocked"}
      <path
        d="M2 4h6M12 4h6M22 4h10"
        stroke="currentColor"
        stroke-width="1.4"
        stroke-linecap="round"
      />
    {:else}
      <path
        d="M2 4h30"
        stroke="currentColor"
        stroke-width="1.4"
        stroke-linecap="round"
      />
    {/if}
    <path
      d="m29 1.4 3.6 2.6-3.6 2.6"
      stroke="currentColor"
      stroke-width="1.4"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</div>

<style>
  .connector {
    color: var(--border);
  }
  .connector[data-tone="live"] {
    color: var(--primary);
  }
  .connector[data-tone="blocked"] {
    color: var(--warning);
  }
</style>
