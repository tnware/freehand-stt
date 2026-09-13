<script lang="ts">
  import type { Model } from "$bindings/managedruntime";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import {
    catalogGroups,
    modelSize,
    runtimePresentation,
  } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import { Badge } from "$lib/components/ui/badge";
  import { Switch } from "$lib/components/ui/switch";
  import * as Dialog from "$lib/components/ui/dialog";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import CpuIcon from "@lucide/svelte/icons/cpu";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import CheckIcon from "@lucide/svelte/icons/check";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import TrashIcon from "@lucide/svelte/icons/trash-2";
  import AlertCircleIcon from "@lucide/svelte/icons/circle-alert";

  let {
    runtime,
    disabled = false,
    workBusy = false,
    onAction,
    onManual,
  }: {
    runtime: ManagedRuntimeState;
    disabled?: boolean;
    workBusy?: boolean;
    onAction: (action: () => void) => void;
    onManual: () => void;
  } = $props();

  const uid = $props.id();
  const status = $derived(runtime.status);
  const view = $derived(runtimePresentation(status));
  const groups = $derived(catalogGroups(status?.models ?? []));
  const operating = $derived(runtime.busy || !!status?.phase);
  const locked = $derived(
    disabled || workBusy || runtime.loading || operating || !status?.supported,
  );
  const running = $derived(status?.state === "running");
  const problem = $derived(runtime.error || status?.error || "");
  const phaseLabel = $derived(
    (
      {
        install: "Installing runtime",
        download: "Downloading model",
        start: "Starting runtime",
        startup: "Starting runtime",
        catalog: "Refreshing catalog",
        remove_model: "Deleting model",
        remove: "Removing runtime",
        stop: "Stopping runtime",
      } as Record<string, string>
    )[status?.phase ?? ""] ||
      status?.phase ||
      runtime.pending ||
      view.label,
  );
  const selectedName = $derived(
    view.selected?.name ||
      (status?.selectedModel === "nemotron-3.5"
        ? "Nemotron 3.5 Streaming"
        : "Selected speech model"),
  );

  type Confirmation =
    { kind: "manual" } | { kind: "runtime" } | { kind: "model"; model: Model };
  let confirmation = $state<Confirmation | null>(null);
  const confirmationTitle = $derived(
    confirmation?.kind === "model"
      ? `Delete ${confirmation.model.name}?`
      : confirmation?.kind === "runtime"
        ? "Remove local runtime?"
        : "Use your own server?",
  );
  const confirmationDescription = $derived(
    confirmation?.kind === "model"
      ? `This deletes the downloaded model from this PC. ${confirmation.model.id === status?.selectedModel ? "It is your selected model: local transcription will be unavailable until you download it again or select another downloaded model. " : ""}You can download it again later. Your saved server settings are unchanged.`
      : confirmation?.kind === "runtime"
        ? "This stops local speech and deletes the managed runtime and its downloaded models from this PC. Local transcription will be unavailable until you reinstall. Your saved server settings are preserved; there is no automatic fallback."
        : "Disable managed local transcription and return to your saved Voice connection and file transcription settings. Review those settings before transcribing. Downloaded models stay on this PC.",
  );
  const confirmationLabel = $derived(
    confirmation?.kind === "model"
      ? "Delete model"
      : confirmation?.kind === "runtime"
        ? "Remove runtime and models"
        : "Disable local runtime",
  );

  function act(
    action: () => Promise<unknown>,
    destructive: Confirmation | null = null,
  ) {
    if (locked) return;
    onAction(() => {
      lastDestructive = destructive;
      void action();
    });
  }

  function confirmAction() {
    if (!confirmation || locked) return;
    const target = confirmation;
    confirmation = null;
    act(
      () =>
        target.kind === "manual"
          ? runtime.disable()
          : target.kind === "runtime"
            ? runtime.run("Remove")
            : runtime.removeModel(target.model.id),
      target,
    );
  }

  function setRealtime(realtime: boolean) {
    if (!status) return;
    const preferences = {
      enabled: status.enabled,
      model: status.selectedModel,
      realtime,
    };
    act(() => runtime.setPreferences(preferences));
  }

  // Retrying a destructive request is still a destructive action: confirm again.
  function retry() {
    if (lastDestructive) confirmation = lastDestructive;
    else act(() => runtime.retry());
  }
  let lastDestructive = $state<Confirmation | null | undefined>(undefined);

  function requestConfirmation(target: Confirmation) {
    confirmation = target;
  }
</script>

<div class="@container/runtime flex min-w-0 flex-col gap-5">
  <section
    aria-labelledby={`${uid}-intro`}
    class="overflow-hidden rounded-xl border border-hairline bg-card"
  >
    <div class="flex items-start gap-3 px-5 pt-4">
      <div class="mt-0.5 shrink-0 text-primary" aria-hidden="true">
        <CpuIcon class="size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2
            id={`${uid}-intro`}
            class="text-base font-semibold tracking-tight"
          >
            Live transcription on this PC
          </h2>
          {#if status?.enabled}<Badge variant="secondary"
              >Selected for Voice & files</Badge
            >{/if}
        </div>
        <p class="mt-1 text-[13px] text-muted-foreground">
          {selectedName} · {status?.realtime
            ? "Realtime dictation"
            : "Completed transcription"}
        </p>
      </div>
      {#if status?.supported}<span
          role="status"
          class="shrink-0 text-xs text-muted-foreground">{view.label}</span
        >{/if}
    </div>

    {#if status && !status.supported}
      <div class="px-5 pb-5 pt-4">
        <p class="text-sm font-medium">
          Local runtime is available on Windows only.
        </p>
        <p class="mt-1 text-[13px] leading-relaxed text-muted-foreground">
          Use an OpenAI-compatible speech server instead. Your server can run
          locally, on your network, or with a hosted provider.
        </p>
        <Button variant="outline" class="mt-4" onclick={onManual}
          >Configure my own server</Button
        >
      </div>
    {:else if !status}
      <div
        class="flex flex-wrap items-center justify-between gap-3 px-5 py-5"
        role="status"
      >
        <p class="text-sm text-muted-foreground">
          {runtime.loading
            ? "Reading local runtime status…"
            : "Local runtime status is unavailable."}
        </p>
        <Button
          variant="outline"
          disabled={runtime.loading || disabled}
          onclick={() => runtime.load()}>Refresh status</Button
        >
      </div>
    {:else}
      <div
        class="px-5 pb-4 pt-3"
        role="region"
        aria-label="Local speech controls"
      >
        <p class="text-[13px] text-muted-foreground">
          {operating
            ? phaseLabel
            : !status.enabled
              ? "Use recommended Nemotron 3.5 with realtime enabled. Downloads remain explicit."
              : !view.installed
                ? "First, install NeMo-Speech.cpp on this PC."
                : !view.selected?.installed
                  ? `Next, download ${selectedName}${view.selected?.sizeBytes ? ` (${modelSize(view.selected.sizeBytes)})` : ""}. Or choose another model below.`
                  : running
                    ? "Ready for Voice and audio files."
                    : "Your model is downloaded. Start the runtime to transcribe."}
        </p>
        <div class="mt-3 flex flex-wrap items-center gap-2">
          {#if operating}
            <Button
              variant="outline"
              disabled={disabled || runtime.pending === "Cancelling"}
              onclick={() => runtime.cancel()}
              >{runtime.pending === "Cancelling"
                ? "Cancelling…"
                : "Cancel operation"}</Button
            >
          {:else if !status.enabled}
            <Button
              disabled={locked}
              onclick={() => act(() => runtime.enable())}
              >Enable local transcription</Button
            >
          {:else if !view.installed}
            <Button
              disabled={locked}
              onclick={() => act(() => runtime.run("Install"))}
              ><DownloadIcon data-icon="inline-start" />Install runtime</Button
            >
          {:else if !view.selected?.installed}
            <Button
              disabled={locked || !view.selected}
              onclick={() =>
                act(() => runtime.downloadModel(status.selectedModel))}
              ><DownloadIcon data-icon="inline-start" />Download selected model</Button
            >
          {:else if running}
            <Button
              variant="outline"
              disabled={locked}
              onclick={() => act(() => runtime.run("Stop"))}
              ><SquareIcon data-icon="inline-start" />Stop runtime</Button
            >
          {:else}
            <Button
              disabled={locked}
              onclick={() => act(() => runtime.run("Start"))}
              ><PlayIcon data-icon="inline-start" />Start runtime</Button
            >
          {/if}
          <Button
            variant="ghost"
            disabled={locked}
            onclick={() =>
              status.enabled
                ? requestConfirmation({ kind: "manual" })
                : onManual()}>Use my own server</Button
          >
        </div>
        {#if operating}
          <div class="mt-3">
            <div
              role="status"
              class="mb-1.5 flex items-center justify-between gap-2 text-xs text-muted-foreground"
            >
              <span class="flex items-center gap-2"
                ><LoaderCircleIcon
                  class="size-3.5 motion-safe:animate-spin"
                  aria-hidden="true"
                />{phaseLabel}</span
              >{#if view.percent !== null}<span class="figure"
                  >{view.percent}%</span
                >{/if}
            </div>
            <progress
              aria-label="Runtime operation"
              max="100"
              value={view.percent ?? undefined}
              class="h-1.5 w-full overflow-hidden rounded-full accent-primary"
            ></progress>
          </div>
        {/if}
        <p class="mt-3 text-xs text-muted-foreground">
          No automatic fallback to your saved server. Cleanup and speech
          playback keep their own connections.
        </p>
      </div>
    {/if}
  </section>

  {#if problem}
    <div
      class="flex gap-3 rounded-xl border border-destructive/25 bg-destructive/5 p-4"
      role="alert"
    >
      <AlertCircleIcon
        class="mt-0.5 size-4 shrink-0 text-destructive"
        aria-hidden="true"
      />
      <div class="min-w-0 flex-1">
        <p class="text-sm font-medium">Local runtime needs attention</p>
        <p
          class="mt-1 break-words text-[13px] leading-relaxed text-muted-foreground"
        >
          {problem}
        </p>
        <div class="mt-3 flex flex-wrap gap-2">
          {#if runtime.canRetry && lastDestructive !== undefined}<Button
              variant="outline"
              size="sm"
              disabled={locked}
              onclick={retry}>Retry operation</Button
            >{/if}<Button
            variant="ghost"
            size="sm"
            disabled={disabled || runtime.loading || operating}
            onclick={() => runtime.load()}>Refresh status</Button
          >
        </div>
      </div>
    </div>
  {/if}

  {#if status?.supported}
    <p class="px-1 text-xs leading-relaxed text-muted-foreground">
      Changes apply immediately. Your saved server connections are preserved.
    </p>
    {#if workBusy}<p
        role="status"
        class="rounded-lg bg-muted/50 px-4 py-3 text-[13px] text-muted-foreground"
      >
        Finish active speech work before changing the local runtime.
      </p>{/if}

    <details class="rounded-lg border border-hairline">
      <summary class="cursor-pointer px-4 py-3 text-sm font-medium"
        >Runtime options <span
          class="ml-2 text-xs font-normal text-muted-foreground"
          >{status.backend || "Automatic acceleration"}{status.version
            ? ` · NeMo ${status.version}`
            : ""}</span
        ></summary
      >
      <SettingsCard>
        <SettingRow
          title="Runtime files"
          description={view.installed
            ? `NeMo-Speech.cpp${status.version ? ` · v${status.version}` : ""}. Managed files stay in Freehand app data.`
            : "Downloads the official, pinned runtime into Freehand app data. Nothing is bundled or installed system-wide."}
        >
          <div class="flex flex-wrap gap-2">
            {#if view.installed}<Button
                variant="ghost"
                disabled={locked}
                onclick={() => requestConfirmation({ kind: "runtime" })}
                ><TrashIcon data-icon="inline-start" />Remove runtime</Button
              >{/if}
          </div>
        </SettingRow>
        <SettingRow
          title="Acceleration"
          description="CPU or GPU is selected automatically from supported hardware. No driver setup or custom launch flags in Freehand."
        >
          {#snippet control()}<Badge variant="outline"
              >{status.backend || "Automatic"}</Badge
            >{/snippet}
        </SettingRow>
        <SettingRow
          controlID={`${uid}-realtime`}
          title="Realtime dictation"
          description={view.selected && !view.selected.realtime
            ? "The selected model supports completed speech only. Choose a realtime-capable model to show text as you speak."
            : "Show text as you speak with a compatible model. Turn off to transcribe completed recordings instead. This preference applies immediately."}
        >
          {#snippet control()}<Switch
              id={`${uid}-realtime`}
              aria-label="Realtime dictation"
              checked={status.realtime}
              disabled={locked || !status.enabled || !view.selected?.realtime}
              onCheckedChange={setRealtime}
            />{/snippet}
        </SettingRow>
        {#if status.selectedModel !== "nemotron-3.5" || !status.realtime}<div
            class="px-4 pb-4"
          >
            <Button
              variant="outline"
              disabled={locked}
              onclick={() => act(() => runtime.enable())}
              >Use recommended realtime setup</Button
            >
          </div>{/if}
      </SettingsCard>
    </details>

    <section
      aria-labelledby={`${uid}-catalog`}
      class="flex min-w-0 flex-col gap-3"
    >
      <div class="flex flex-wrap items-center justify-between gap-3 px-1">
        <div>
          <h3 id={`${uid}-catalog`} class="text-sm font-semibold">
            Speech models
          </h3>
          <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
            Catalog browsing is metadata-only. Only Download fetches model
            files.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={locked}
          onclick={() => act(() => runtime.run("RefreshCatalog"))}
          ><RefreshCwIcon data-icon="inline-start" />Refresh catalog</Button
        >
      </div>
      {#if !status.models?.length}
        <div class="rounded-xl border border-dashed border-hairline px-5 py-6">
          <p class="text-sm font-medium">
            {view.installed
              ? "No models listed yet"
              : "Install the runtime to browse models"}
          </p>
          <p class="mt-1 text-[13px] leading-relaxed text-muted-foreground">
            {view.installed
              ? "Refresh the catalog to read the runtime’s model index. This does not download or load any models."
              : "After installation, the catalog shows supported speech models and their download sizes."}
          </p>
        </div>
      {:else}
        {#if running}<p class="px-1 text-xs text-muted-foreground">
            Stop the runtime before downloading another model.
          </p>{/if}
        {#each groups as group (group.label)}
          {#if group.models.length}
            <div class="flex min-w-0 flex-col gap-2">
              <h4 class="px-1 text-xs font-medium text-muted-foreground">
                {group.label}
              </h4>
              {#each group.models as model (model.id)}
                <article
                  aria-label={model.name}
                  class={model.id === status.selectedModel
                    ? "min-w-0 rounded-xl border border-primary/30 bg-card p-4"
                    : "min-w-0 rounded-xl border border-hairline bg-card p-4"}
                >
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0 flex-[1_1_12rem]">
                      <div class="flex flex-wrap items-center gap-2">
                        <h5 class="break-words text-sm font-semibold">
                          {model.name}
                        </h5>
                        {#if model.recommended}<Badge variant="secondary"
                            >Recommended</Badge
                          >{/if}{#if model.id === status.selectedModel}<Badge
                            variant="outline">Selected</Badge
                          >{/if}
                      </div>
                      <p
                        class="mt-1.5 break-words text-[13px] leading-relaxed text-muted-foreground"
                      >
                        {model.description}
                      </p>
                    </div>
                    <div class="flex shrink-0 flex-wrap items-center gap-2">
                      {#if model.installed}<Button
                          variant="outline"
                          size="sm"
                          disabled={locked || model.id === status.selectedModel}
                          onclick={() => act(() => runtime.useModel(model.id))}
                          >{model.id === status.selectedModel
                            ? "Selected"
                            : "Use model"}</Button
                        ><Button
                          variant="ghost"
                          size="sm"
                          disabled={locked}
                          onclick={() =>
                            requestConfirmation({ kind: "model", model })}
                          >Delete</Button
                        >
                      {:else}<Button
                          variant="outline"
                          size="sm"
                          disabled={locked || !view.installed || running}
                          onclick={() =>
                            act(() => runtime.downloadModel(model.id))}
                          ><DownloadIcon
                            data-icon="inline-start"
                          />Download</Button
                        >{/if}
                    </div>
                  </div>
                  <div
                    class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
                  >
                    <span
                      >{model.realtime
                        ? "Realtime + completed speech"
                        : "Completed speech"}</span
                    ><span class="figure">{modelSize(model.sizeBytes)}</span
                    >{#if model.installed}<span class="flex items-center gap-1"
                        ><CheckIcon
                          class="size-3"
                          aria-hidden="true"
                        />Downloaded</span
                      >{/if}
                  </div>
                </article>
              {/each}
            </div>
          {/if}
        {/each}
      {/if}
    </section>
  {/if}
</div>

<Dialog.Root
  open={confirmation !== null}
  onOpenChange={(open) => {
    if (!open) confirmation = null;
  }}
>
  <Dialog.Content showCloseButton={false}>
    <Dialog.Header
      ><Dialog.Title>{confirmationTitle}</Dialog.Title><Dialog.Description
        >{confirmationDescription}</Dialog.Description
      ></Dialog.Header
    >
    <Dialog.Footer
      ><Button
        variant="outline"
        onclick={() => {
          confirmation = null;
        }}>Keep current setup</Button
      ><Button
        variant={confirmation?.kind === "manual" ? "default" : "destructive"}
        disabled={locked}
        onclick={confirmAction}>{confirmationLabel}</Button
      ></Dialog.Footer
    >
  </Dialog.Content>
</Dialog.Root>
