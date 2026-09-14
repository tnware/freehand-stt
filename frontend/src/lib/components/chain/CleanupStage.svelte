<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import { Switch } from "$lib/components/ui/switch";
  import Stage, { type StageTone } from "./Stage.svelte";
  import StagePick from "./StagePick.svelte";

  let {
    ordinal = "03",
    enabled,
    model,
    source = "",
    profile = "",
    busy = false,
    unused = false,
    onToggle,
    onOpen,
  }: {
    ordinal?: string;
    enabled: boolean;
    model: string;
    source?: string;
    /** The cleanup preset in force, when one is selected. */
    profile?: string;
    busy?: boolean;
    /** Text to speech has no transcript to clean, so the stage dims in place. */
    unused?: boolean;
    onToggle: (next: boolean) => void;
    onOpen: () => void;
  } = $props();

  const tone = $derived<StageTone>(
    unused ? "off" : busy ? "busy" : enabled ? "ok" : "off",
  );
</script>

<Stage ordinal={ordinal} label="Clean up" {tone} {unused}>
  {#snippet control()}
    {#if unused}
      <span class="text-[10px] text-ink-disabled">n/a</span>
    {:else if busy}
      <LoaderCircleIcon
        class="size-3.5 animate-spin text-muted-foreground"
        aria-hidden="true"
      />
    {:else}
      <Switch
        checked={enabled}
        onCheckedChange={onToggle}
        aria-label="Clean up transcripts"
        class="scale-[0.8]"
      />
    {/if}
  {/snippet}

  {#if unused}
    <p class="px-1 text-[13px] font-medium text-ink-disabled">Not used here</p>
    <p class="px-1 font-mono text-[10px] text-ink-disabled">
      no transcript to clean
    </p>
  {:else}
    <StagePick
      value={enabled ? model || "Not selected" : "Off"}
      meta={enabled ? source : "transcripts pass through unchanged"}
      label="Cleanup model"
      {onOpen}
    />
  {/if}

  {#snippet footer()}
    {#if !unused && enabled && profile}
      <span
        class="truncate rounded-sm border border-border px-1.5 py-0.5 text-[11px] text-muted-foreground"
        >{profile}</span
      >
    {:else}
      <span></span>
    {/if}
  {/snippet}
</Stage>
