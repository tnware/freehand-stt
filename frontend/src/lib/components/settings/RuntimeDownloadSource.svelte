<script lang="ts">
  import type { RuntimeSource } from "$bindings/managedruntime";
  import { Browser } from "@wailsio/runtime";
  import { backendLabel, modelSize } from "$lib/utils/managedRuntime";

  let { source, backend = "" }: { source: RuntimeSource; backend?: string } =
    $props();
  let linkError = $state(false);
  const artifacts = $derived(
    (source.artifacts ?? []).filter(
      (artifact) => !backend || artifact.backend === backend,
    ),
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

<div class="min-w-0 space-y-1 text-xs text-foreground/80">
  <p class="flex flex-wrap items-baseline gap-x-1.5 gap-y-1">
    <span>Source</span>
    <a
      class="break-all text-primary underline underline-offset-2 hover:text-primary/80"
      href={source.repositoryURL}
      onclick={(event) => void open(event, source.repositoryURL)}
      >{source.repositoryURL.replace("https://github.com/", "")}</a
    >
    <span aria-hidden="true">·</span>
    <a
      class="text-primary underline underline-offset-2 hover:text-primary/80"
      href={source.releaseURL}
      onclick={(event) => void open(event, source.releaseURL)}
      >Official release</a
    >
  </p>
  <details class="group">
    <summary
      class="w-fit cursor-pointer py-1 text-foreground/80 hover:text-foreground"
      >Binary download details</summary
    >
    <div class="mt-2 space-y-3 border-l border-border pl-3">
      {#each artifacts as artifact (artifact.url)}
        <div class="min-w-0 space-y-1">
          <p class="font-medium text-foreground">
            {backendLabel(artifact.backend)} · {artifact.os} / {artifact.architecture}
            · {modelSize(artifact.sizeBytes)}
          </p>
          <p class="break-all font-mono">{artifact.filename}</p>
          <p class="break-all font-mono text-[11px]">
            <span class="font-sans">SHA-256 </span>{artifact.sha256}
          </p>
        </div>
      {/each}
    </div>
  </details>
  {#if linkError}<p role="alert" class="text-destructive">
      Could not open the source in your browser.
    </p>{/if}
</div>
