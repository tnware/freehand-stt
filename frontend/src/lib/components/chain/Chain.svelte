<script lang="ts">
  import type { Snippet } from "svelte";
  import Connector from "./Connector.svelte";

  export type ChainLink = { label: string; tone?: "idle" | "live" | "blocked" };

  let {
    links,
    stages,
    label = "Signal chain",
  }: {
    /** One fewer than the stages: what flows out of each stage but the last. */
    links: ChainLink[];
    stages: Snippet[];
    label?: string;
  } = $props();
</script>

<!--
  The chain is an ordered list because the order is the meaning. Below the
  width where four cards and their connectors fit, the connectors drop out and
  the stages reflow: the ordinals in each header keep the sequence readable,
  which is why they are rendered as text rather than drawn.
-->
<ol class="chain" aria-label={label}>
  {#each stages as stage, index (index)}
    <li class="chain-stage">{@render stage()}</li>
    {#if index < stages.length - 1 && links[index]}
      <li class="chain-link">
        <Connector
          label={links[index].label}
          tone={links[index].tone ?? "idle"}
        />
      </li>
    {/if}
  {/each}
</ol>

<style>
  .chain {
    display: flex;
    align-items: stretch;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .chain-stage {
    display: flex;
    min-width: 0;
    flex: 1;
    /* Equal, fixed height across the row: a stage whose state has nothing to
       show must not make the chain ragged. */
    min-height: 18.5rem;
  }
  .chain-link {
    display: flex;
    flex: 0 0 2.5rem;
  }

  /* Four stages plus three connectors need roughly 1000px before the cards
     start squeezing their own content. Below that the connectors go and the
     chain becomes a grid; the stage ordinals carry the sequence. */
  @container (max-width: 1000px) {
    .chain {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 0.5rem;
    }
    .chain-link {
      display: none;
    }
  }
  @container (max-width: 620px) {
    .chain {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
