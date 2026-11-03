<script lang="ts">
  import type {
    ProviderDescriptor,
    InstanceStatus,
  } from "$bindings/managedruntime";
  import { localSpeechSetup } from "$lib/utils/localSpeechSetup";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import Disclosure from "$lib/components/common/Disclosure.svelte";
  import type { Snippet } from "svelte";
  import CheckIcon from "@lucide/svelte/icons/check";
  import AudioLinesIcon from "@lucide/svelte/icons/audio-lines";
  import ListChecksIcon from "@lucide/svelte/icons/list-checks";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { SettingsSectionID } from "$lib/navigation";
  import type { Readiness, ReadinessStep } from "$lib/utils/readiness";
  import { nextReadinessAction } from "$lib/utils/readinessPresentation";

  let {
    readiness,
    serverControls,
    providers = [],
    instances = [],
    onManageRuntime,
    onChooseServer,
    task = "voice",
    embedded = false,
    testing = false,
    completing = false,
    saving = false,
    onTestConnection,
    onComplete,
    onDismiss,
    onOpenSettings,
  }: {
    readiness: Readiness;
    serverControls?: Snippet;
    providers?: ProviderDescriptor[];
    instances?: InstanceStatus[];
    onManageRuntime?: (providerID?: string, instanceID?: string) => void;
    onChooseServer?: () => void;
    task?: "voice" | "file";
    embedded?: boolean;
    testing?: boolean;
    completing?: boolean;
    saving?: boolean;
    onTestConnection: () => void;
    onComplete: () => void;
    onDismiss: () => void;
    onOpenSettings: (section: SettingsSectionID) => void;
  } = $props();
  const configuredServer = $derived(
    readiness.steps.find(
      (step) =>
        step.id === "server" &&
        (readiness.connectionSelected ||
          step.status === "complete" ||
          step.settingsSection === "local-runtime"),
    ),
  );
  const localSetup = $derived(localSpeechSetup(providers, instances));
  const existingSetup = $derived(
    !configuredServer ? localSetup.existing : null,
  );
  const localRecommendation = $derived(localSetup.recommendation);
  const action = $derived(nextReadinessAction(readiness));
  const remaining = $derived(
    readiness.steps.filter(
      (step) =>
        step.status !== "complete" && (readiness.initialSetup || step.blocking),
    ),
  );
  const complete = $derived(
    readiness.steps.filter((step) => step.status === "complete"),
  );
  const busy = $derived(testing || completing || saving);
  const connectionNeedsAttention = $derived(
    remaining.some((step) =>
      ["server", "credential", "connection"].includes(step.id),
    ),
  );
  const nextIssue = $derived(
    action.kind === "settings"
      ? remaining.find((step) => step.id === action.step)
      : remaining[0],
  );
  const recoveryTitle = $derived(
    action.kind === "settings"
      ? {
          server: "Choose a transcription connection",
          credential: "Check your authentication",
          microphone: "Check your microphone",
          shortcut: "Choose a recording shortcut",
          connection: "Check your connection",
        }[action.step]
      : "Connection needs attention",
  );
  const title = $derived(
    readiness.initialSetup
      ? task === "file"
        ? "Set up file transcription"
        : "Set up dictation"
      : recoveryTitle,
  );
  function proceed() {
    if (busy) return;
    if (action.kind === "complete") onComplete();
    else if (action.kind === "check") onTestConnection();
    else if (action.kind === "settings") {
      if (action.section === "local-runtime" && onManageRuntime)
        onManageRuntime(undefined, readiness.managedInstanceID);
      else onOpenSettings(action.section);
    }
  }
</script>

<section
  class="readiness min-h-0 flex-1 overflow-y-auto"
  aria-label={readiness.initialSetup ? "First-run setup" : "Task recovery"}
>
  {#if !embedded && !readiness.initialSetup}
    <PaneHeader
      icon={ListChecksIcon}
      {title}
      summary={task === "file"
        ? "Audio-file transcription"
        : "Voice transcription"}
    />
  {/if}
  {#if readiness.initialSetup}
    <div class="welcome">
      <header class="welcome-header">
        <div class="welcome-mark" aria-hidden="true">
          <AudioLinesIcon class="size-5" />
        </div>
        <div class="min-w-0 space-y-2">
          <h2 class="welcome-title">{title}</h2>
          <p class="welcome-description">
            {configuredServer?.settingsSection === "local-runtime" &&
            configuredServer.status !== "complete"
              ? "Finish setup for your selected local model."
              : existingSetup
                ? "Continue setting up your local runtime and model."
                : configuredServer
                  ? task === "voice"
                    ? "Check your connection and recording settings."
                    : "Check your connection before transcribing audio files."
                  : task === "voice"
                    ? "Install a local speech model or connect to a transcription service."
                    : "Choose a model for transcribing audio files."}
          </p>
        </div>
      </header>
      {#if !(onManageRuntime && !configuredServer && action.kind === "settings" && action.step === "server")}
        <section class="welcome-action" aria-label="Next setup step">
          <p class="content-meta" aria-live="polite">
            {readiness.canComplete
              ? "Connection checks passed."
              : action.kind === "settings" &&
                  action.step === "server" &&
                  !configuredServer
                ? "Choose local processing or an existing server."
                : action.kind === "check"
                  ? "Check the connection. No audio will be sent."
                  : (nextIssue?.detail ?? "Choose a transcription connection.")}
          </p>
          <div class="mt-4 flex flex-wrap items-center gap-2">
            {@render primaryAction()}
          </div>
        </section>
      {/if}
      {#if configuredServer}
        <section
          class="welcome-action"
          aria-label="Current transcription connection"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="min-w-0 flex-1">
              <p class="content-section-title">Current connection and model</p>
              <p class="content-meta mt-1 break-words">
                {configuredServer.summary ?? configuredServer.detail}
              </p>
            </div>
            <Button
              variant="outline"
              disabled={busy}
              onclick={() =>
                onOpenSettings(
                  task === "voice" ? "voice-transcription" : "server",
                )}>Change</Button
            >
          </div>
        </section>
      {/if}
      {#if onManageRuntime && existingSetup}
        <section
          class="welcome-action"
          aria-label="Continue local speech setup"
        >
          <p class="content-kicker">{existingSetup.context.heading}</p>
          <h3 class="content-section-title mt-1">
            {existingSetup.context.next}
          </h3>
          <p class="content-meta mt-2 break-words">
            {existingSetup.context.model.label}: {existingSetup.context.model
              .name}
          </p>
          {#if existingSetup.context.detail}<p class="content-meta mt-1">
              {existingSetup.context.detail}
            </p>{/if}
          <Button
            class="mt-3"
            disabled={busy}
            onclick={() =>
              onManageRuntime?.(
                existingSetup.provider.id,
                existingSetup.instance.instance.id,
              )}
          >
            {existingSetup.instance.status.state === "running"
              ? "Review and use local model"
              : "Continue local setup"}
          </Button>
        </section>
      {/if}
      {#if onManageRuntime && !configuredServer && !existingSetup}
        <section class="local-choice" aria-label="Local speech setup">
          <div class="local-heading">
            <h3 class="content-section-title">
              {localSetup.availability === "unavailable"
                ? "Connect to a speech server"
                : "Run on this computer"}
            </h3>
            {#if localRecommendation}<span class="recommendation-label"
                >Recommended</span
              >{/if}
          </div>
          {#if localRecommendation}
            <div class="model-identity">
              <div class="model-icon">
                <ProviderIcon
                  profile={localRecommendation.provider.id}
                  size={32}
                />
              </div>
              <div class="min-w-0">
                <p class="model-name">{localRecommendation.model.name}</p>
                <p class="content-meta mt-1">
                  {localRecommendation.provider.name}
                </p>
                <p class="content-meta mt-2">
                  {localRecommendation.model.description}
                </p>
              </div>
            </div>
          {:else}
            <p class="content-meta mt-3">
              {localSetup.availability === "unavailable"
                ? "Built-in transcription is not available on this platform. Use your own server or a compatible provider."
                : localSetup.availability === "unknown"
                  ? "Check available local engines and platform requirements."
                  : "View supported local engines and models."}
            </p>
          {/if}
          {#if localSetup.availability === "available"}
            <p class="content-meta mt-2">
              Download a speech engine and model, then transcribe locally.
              You’ll review the download before it starts.
            </p>
          {/if}
          <div class="local-actions">
            {#if localSetup.availability === "unavailable" && onChooseServer}<Button
                disabled={busy}
                onclick={onChooseServer}>Use a server</Button
              >{/if}
            <Button
              variant={localSetup.availability === "available" &&
              action.kind === "settings" &&
              action.step === "server"
                ? "default"
                : "outline"}
              disabled={busy}
              onclick={() =>
                onManageRuntime?.(localRecommendation?.provider.id)}
              >{localRecommendation
                ? "Set up local speech"
                : "Browse local engines"}</Button
            >
          </div>
          {#if localRecommendation}<button
              type="button"
              class="mt-3 text-xs text-muted-foreground underline-offset-4 hover:underline"
              disabled={busy}
              onclick={() => onManageRuntime?.()}
              >See all local engines and models</button
            >{/if}
        </section>
      {/if}
      {#if !configuredServer && !existingSetup && onManageRuntime && onChooseServer && localSetup.availability !== "unavailable"}
        <div class="server-alternative">
          <div class="min-w-0">
            <p class="content-section-title">Connect to a service</p>
            <p class="content-meta mt-1">
              Connect to your own endpoint or a provider.
            </p>
          </div>
          <Button variant="outline" disabled={busy} onclick={onChooseServer}
            >Use a server</Button
          >
        </div>
      {/if}
      <div class="welcome-options">
        {#if (configuredServer || existingSetup) && (onManageRuntime || onChooseServer)}
          <Disclosure title="Other connection options">
            <div class="flex flex-wrap gap-2 py-2">
              {#if onManageRuntime}<Button
                  variant="outline"
                  disabled={busy}
                  onclick={() => onManageRuntime?.()}
                  >Browse local engines</Button
                >{/if}
              {#if onChooseServer}<Button
                  variant="outline"
                  disabled={busy}
                  onclick={onChooseServer}>Use a server</Button
                >{/if}
            </div>
          </Disclosure>
        {/if}
        {#if serverControls}
          <Disclosure
            title="Connection and model"
            description="On your computer, your server, or a provider."
          >
            <div class="py-2">{@render serverControls()}</div>
          </Disclosure>
        {/if}
        <Disclosure title="Setup details">
          <p class="content-meta py-2">
            Connection checks read server metadata only. Your first
            transcription confirms that the selected model can run.
          </p>
          <div class="divide-y divide-hairline">
            {#each readiness.steps as step (step.id)}{@render readyStep(
                step,
              )}{/each}
          </div>
        </Disclosure>
      </div>
    </div>
  {:else}
    <div class="flex w-full max-w-[640px] flex-col gap-4 px-3 py-4">
      {#if embedded}<h2 class="content-section-title">{title}</h2>{/if}
      <section class="settings-group" aria-label="Next setup step">
        <div class="space-y-4 py-3">
          {#each remaining as step (step.id)}
            <div class="flex items-start gap-2.5">
              {#if step.status === "attention"}<CircleAlertIcon
                  class="mt-0.5 size-4 shrink-0 text-warning"
                />
              {:else if step.id === "microphone"}<LoaderCircleIcon
                  class="mt-0.5 size-4 shrink-0 animate-spin text-muted-foreground motion-reduce:animate-none"
                />{/if}
              <div class="min-w-0 flex-1">
                <p class="content-value">{step.label}</p>
                <p class="content-meta mt-1 break-words">{step.detail}</p>
              </div>
              {#if step.settingsSection && !(action.kind === "settings" && action.step === step.id) && step.status === "attention"}
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busy}
                  aria-label={`Review ${step.label} settings`}
                  onclick={() => onOpenSettings(step.settingsSection!)}
                  >Review</Button
                >
              {/if}
            </div>
          {/each}
        </div>
        <div
          class="flex flex-wrap items-center gap-2 border-t border-hairline py-3"
        >
          {@render primaryAction()}
          <Button variant="ghost" size="sm" disabled={busy} onclick={onDismiss}
            >Back to workspace</Button
          >
        </div>
      </section>
      {#if complete.length}
        <Disclosure
          title={`${complete.length} ${complete.length === 1 ? "check ready" : "checks ready"}`}
          icon={CheckIcon}
          class="border-t border-hairline"
        >
          <div class="divide-y divide-hairline">
            {#each complete as step (step.id)}{@render readyStep(step)}{/each}
          </div>
        </Disclosure>
      {/if}
      {#if connectionNeedsAttention && serverControls}
        <Disclosure
          title="Connection and model"
          class="border-t border-hairline"
        >
          {@render serverControls()}
        </Disclosure>
      {/if}
    </div>
  {/if}
</section>

{#snippet primaryAction()}
  <Button
    variant={readiness.initialSetup &&
    onManageRuntime &&
    action.kind === "settings" &&
    action.step === "server"
      ? "outline"
      : "default"}
    size={readiness.initialSetup ? "default" : "sm"}
    disabled={busy || action.kind === "wait"}
    onclick={proceed}
    class="min-w-0"
  >
    {#if busy || action.kind === "wait"}<LoaderCircleIcon
        class="size-4 animate-spin motion-reduce:animate-none"
      />{/if}
    {testing
      ? "Checking connection…"
      : completing
        ? "Finishing…"
        : saving
          ? "Saving settings…"
          : action.label}
  </Button>
{/snippet}

{#snippet readyStep(step: ReadinessStep)}
  <div class="flex items-start gap-3 py-3">
    <div class="min-w-0 flex-1">
      <p class="content-value">{step.label}</p>
      {#if step.id === "shortcut"}<div class="mt-1">
          <ShortcutKeys value={step.detail} label="Toggle recording shortcut" />
        </div>
      {:else}<p class="content-meta mt-1 break-words">{step.detail}</p>{/if}
    </div>
    {#if step.settingsSection}<Button
        variant="ghost"
        size="sm"
        disabled={busy}
        aria-label={`Review ${step.label} settings`}
        onclick={() => onOpenSettings(step.settingsSection!)}>Review</Button
      >{/if}
  </div>
{/snippet}

<style>
  .welcome {
    width: 100%;
    max-width: 42rem;
    margin-inline: auto;
    padding: clamp(1rem, 3cqw, 1.75rem) 1rem;
  }
  .welcome-header {
    display: flex;
    gap: 0.875rem;
    align-items: flex-start;
  }
  .welcome-mark {
    display: flex;
    flex-shrink: 0;
    width: 2.25rem;
    height: 2.25rem;
    align-items: center;
    justify-content: center;
    border-radius: 0.75rem;
    color: var(--accent-text);
    background: var(--subtle-fill);
  }
  .welcome-title {
    font-size: clamp(1.25rem, 3cqw, 1.5rem);
    line-height: 1.25;
    font-weight: 600;
    letter-spacing: -0.025em;
    text-wrap: balance;
  }
  .welcome-description {
    max-width: 52ch;
    font-size: 13px;
    line-height: 1.6;
    color: var(--muted-foreground);
  }
  .local-choice {
    margin-top: 1.25rem;
    padding: 1rem;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    background: var(--subtle-fill);
  }
  .local-heading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .recommendation-label {
    border: 1px solid var(--hairline);
    border-radius: 0.375rem;
    padding: 0.125rem 0.375rem;
    color: var(--accent-text);
    font-size: 10px;
    font-weight: 500;
  }
  .model-identity {
    display: flex;
    align-items: flex-start;
    gap: 0.875rem;
    margin-block: 1.125rem;
  }
  .model-icon {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
    background: var(--background);
    border: 1px solid var(--hairline);
    border-radius: 0.75rem;
  }
  .model-name {
    font-size: 16px;
    font-weight: 600;
    line-height: 1.4;
    overflow-wrap: anywhere;
  }
  .local-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 1rem;
  }
  .server-alternative {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 1rem 0.25rem;
  }
  .welcome-action {
    margin-block: 1.25rem;
  }
  .welcome-options {
    margin-top: 0.75rem;
    border-top: 1px solid var(--hairline);
  }
</style>
