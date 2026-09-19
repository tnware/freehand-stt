<script lang="ts">
  import DownloadDetails from "$lib/components/common/DownloadDetails.svelte";
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
        class="source-link"
        href={source.repositoryURL}
        onclick={(event) => void open(event, event.currentTarget.href)}
        >{source.repository}</a
      >
      <span> · Freehand download</span>
    {:else}
      Source: {source.repository}
    {/if}
  </p>
  {#if source.companions?.length}<p>
      Includes {source.companions.length} companion files, downloaded and verified
      with the model.
    </p>{/if}
  <DownloadDetails title="Model download details">
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
        <dd class="break-all font-mono text-xs">{source.sha256}</dd>{/if}
    </dl>
    {#each (source.companions ?? []).filter((companion) => companion !== null) as companion (companion.filename)}
      <dl
        class="mt-3 grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5 border-t border-hairline pt-3 [&>dt]:text-muted-foreground"
      >
        <dt>Companion file</dt>
        <dd class="break-all font-mono">{companion.filename}</dd>
        <dt>Repository</dt>
        <dd class="break-all">{companion.repository}</dd>
        <dt>Revision</dt>
        <dd class="break-all font-mono">{companion.revision}</dd>
        <dt>SHA-256</dt>
        <dd class="break-all font-mono text-xs">{companion.sha256}</dd>
      </dl>
    {/each}
    {#if delegated}<p class="mt-2">
        Model details from NeMo’s bundled index.
      </p>{/if}
  </DownloadDetails>
  {#if linkError}<p role="alert" class="text-destructive">
      Could not open the source in your browser.
    </p>{/if}
</div>
