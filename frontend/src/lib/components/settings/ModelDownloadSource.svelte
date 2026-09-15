<script lang="ts">
  import {
    ModelAcquisitionMethod,
    type ModelSource,
  } from "$bindings/managedruntime";
  import { Browser } from "@wailsio/runtime";

  let {
    source,
    description = "",
  }: { source: ModelSource; description?: string } = $props();
  let linkError = $state(false);
  const delegated = $derived(
    source.acquisitionMethod === ModelAcquisitionMethod.NeMoModelManager,
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

<div class="min-w-0 space-y-1 text-xs text-secondary-foreground">
  <p>
    {#if delegated}
      Source: NeMo’s built-in model manager
    {:else if source.repositoryURL}
      Source: <a
        class="break-all text-accent-text underline underline-offset-2 hover:text-accent-text"
        href={source.repositoryURL}
        onclick={(event) => void open(event, event.currentTarget.href)}
        >{source.repository}</a
      >
      <span> · Freehand download</span>
    {:else}
      Source: {source.repository}
    {/if}
  </p>
  <details>
    <summary
      class="w-fit cursor-pointer rounded-sm py-1 font-medium text-accent-text hover:underline focus-visible:outline-ring"
      >Model download details</summary
    >
    {#if description}<p class="mt-3 leading-relaxed">{description}</p>{/if}
    <dl
      class="mt-3 grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5 [&>dt]:text-muted-foreground"
    >
      <dt>Downloaded by</dt>
      <dd>{delegated ? "NeMo model manager" : "Freehand from Hugging Face"}</dd>
      <dt>{delegated ? "Index repository" : "Repository"}</dt>
      <dd class="break-all">{source.repository}</dd>
      {#if source.revision}<dt>Revision</dt>
        <dd class="break-all font-mono">{source.revision}</dd>{/if}
      {#if source.filename}<dt>File</dt>
        <dd class="break-all font-mono">{source.filename}</dd>{/if}
      {#if source.sha256}<dt>SHA-256</dt>
        <dd class="break-all font-mono text-[11px]">{source.sha256}</dd>{/if}
    </dl>
    {#if delegated}<p class="mt-2">
        Model details from NeMo’s bundled index.
      </p>{/if}
  </details>
  {#if linkError}<p role="alert" class="text-destructive">
      Could not open the source in your browser.
    </p>{/if}
</div>
