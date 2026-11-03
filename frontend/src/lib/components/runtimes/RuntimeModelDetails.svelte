<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import {
    ModelAcquisitionMethod,
    type ModelSource,
  } from "$bindings/managedruntime";

  let {
    modelID,
    variant = "",
    source,
  }: {
    modelID: string;
    variant?: string;
    source?: ModelSource | null;
  } = $props();
  let linkError = $state(false);
  const delegated = $derived(
    source?.acquisitionMethod === ModelAcquisitionMethod.NeMoModelManager,
  );
  async function open(event: MouseEvent, url: string) {
    event.preventDefault();
    linkError = false;
    try {
      await Browser.OpenURL(url);
    } catch {
      linkError = true;
    }
  }
</script>

<div class="model-information">
  <dl class="overview">
    <div>
      <dt>Model ID</dt>
      <dd class="font-mono">{modelID}</dd>
      {#if variant}<dd class="mt-1 text-muted-foreground">{variant}</dd>{/if}
    </div>
    {#if source}
      <div>
        <dt>Source</dt>
        <dd>
          {#if delegated}
            NeMo’s built-in model manager
          {:else if source.repositoryURL}
            <a
              class="source-link"
              href={source.repositoryURL}
              onclick={(event) => void open(event, event.currentTarget.href)}
              >{source.repository}</a
            >
          {:else}{source.repository}{/if}
        </dd>
        <dd class="mt-1 text-muted-foreground">
          {delegated
            ? "Model details from NeMo’s bundled index."
            : "Downloaded by Freehand"}
        </dd>
      </div>
      {#if source.filename}
        <div class="file-fact">
          <dt>Model file</dt>
          <dd class="font-mono">{source.filename}</dd>
          {#if source.companions?.length}<dd class="mt-1 text-muted-foreground">
              Includes {source.companions.length} companion files, downloaded and
              verified with the model.
            </dd>{/if}
        </div>
      {/if}
    {/if}
  </dl>
  {#if source}
    <details class="technical-details">
      <summary>
        <ChevronRightIcon class="technical-chevron" aria-hidden="true" />
        <span>Model download details</span>
      </summary>
      <div class="technical-body">
        <dl class="technical-facts">
          <dt>{delegated ? "Index repository" : "Repository"}</dt>
          <dd>{source.repository}</dd>
          {#if source.revision}<dt>Revision</dt>
            <dd class="font-mono">{source.revision}</dd>{/if}
          {#if source.sha256}<dt>SHA-256</dt>
            <dd class="font-mono">{source.sha256}</dd>{/if}
        </dl>
        {#each (source.companions ?? []).filter((companion) => companion !== null) as companion (companion.filename)}
          <div class="companion-file">
            <h4>Companion file</h4>
            <p class="mb-3 break-all font-mono">{companion.filename}</p>
            <dl class="technical-facts">
              <dt>Repository</dt>
              <dd>{companion.repository}</dd>
              <dt>Revision</dt>
              <dd class="font-mono">{companion.revision}</dd>
              <dt>SHA-256</dt>
              <dd class="font-mono">{companion.sha256}</dd>
            </dl>
          </div>
        {/each}
      </div>
    </details>
  {/if}
  {#if linkError}<p role="alert" class="mt-2 text-destructive">
      Could not open the source in your browser.
    </p>{/if}
</div>

<style>
  .model-information {
    min-width: 0;
    padding: 12px 14px;
    color: var(--secondary-foreground);
    background: var(--background);
    font-size: 12px;
    line-height: 1.5;
  }
  .overview {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.5fr);
    gap: 10px 24px;
  }
  .overview > div {
    display: grid;
    grid-template-columns: 66px minmax(0, 1fr);
    align-content: start;
    column-gap: 10px;
  }
  .overview dd {
    grid-column: 2;
  }
  dt,
  h4 {
    margin-bottom: 0;
    color: var(--muted-foreground);
    font-size: 11px;
    font-weight: 500;
  }
  dd {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .file-fact {
    grid-column: 1 / -1;
  }
  .technical-details {
    margin-top: 10px;
  }
  summary {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding-block: 3px;
    color: var(--accent-text);
    cursor: pointer;
    list-style: none;
  }
  summary::-webkit-details-marker {
    display: none;
  }
  summary:hover {
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  summary:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 3px;
  }
  summary :global(.technical-chevron) {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
  }
  details[open] summary :global(.technical-chevron) {
    transform: rotate(90deg);
  }
  .technical-body {
    margin-top: 10px;
    padding-top: 12px;
    border-top: 1px solid var(--hairline);
  }
  .technical-facts {
    display: grid;
    grid-template-columns: 90px minmax(0, 1fr);
    gap: 8px 16px;
  }
  .companion-file {
    margin-top: 14px;
    padding-top: 12px;
    border-top: 1px solid var(--hairline);
  }
  @container (max-width: 480px) {
    .model-information {
      padding: 12px;
    }
    .overview {
      grid-template-columns: minmax(0, 1fr);
      gap: 10px;
    }
  }
  @container (max-width: 320px) {
    .technical-facts {
      grid-template-columns: minmax(0, 1fr);
      gap: 0;
    }
    .technical-facts dd {
      margin-bottom: 10px;
    }
  }
</style>
