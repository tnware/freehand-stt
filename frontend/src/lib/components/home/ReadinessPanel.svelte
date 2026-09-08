<script lang="ts">
  import type { Snippet } from "svelte";
  import CheckIcon from "@lucide/svelte/icons/check";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { SettingsSectionID } from "$lib/navigation";
  import type { Readiness, ReadinessStep } from "$lib/utils/readiness";
  import { nextReadinessAction } from "$lib/utils/readinessPresentation";

  let {
    readiness,
    serverControls,
    task = "voice",
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
    task?: "voice" | "file";
    testing?: boolean;
    completing?: boolean;
    saving?: boolean;
    onTestConnection: () => void;
    onComplete: () => void;
    onDismiss: () => void;
    onOpenSettings: (section: SettingsSectionID) => void;
  } = $props();
  const action = $derived(nextReadinessAction(readiness));
  const remaining = $derived(
    readiness.steps.filter(
      (step) => step.status !== "complete" && (readiness.initialSetup || step.blocking),
    ),
  );
  const complete = $derived(readiness.steps.filter((step) => step.status === "complete"));
  const busy = $derived(testing || completing || saving);
  const connectionNeedsAttention = $derived(
    remaining.some((step) => ["server", "credential", "connection"].includes(step.id)),
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
  function proceed() {
    if (busy) return;
    if (action.kind === "complete") onComplete();
    else if (action.kind === "check") onTestConnection();
    else if (action.kind === "settings") onOpenSettings(action.section);
  }
</script>

<section
  class="readiness min-h-0 flex-1 overflow-y-auto"
  aria-label={readiness.initialSetup ? "First-run setup" : "Task recovery"}
>
  <div class="mx-auto flex w-full max-w-[960px] flex-col gap-5 py-3">
    <header class="space-y-2">
      <p class="text-xs font-medium text-muted-foreground">
        {readiness.initialSetup
          ? "Welcome to Freehand"
          : task === "file"
            ? "Audio-file transcription"
            : "Voice transcription"}
      </p>
      <h2 class="text-2xl font-semibold tracking-tight">
        {readiness.initialSetup ? "Set up voice transcription" : recoveryTitle}
      </h2>
      <p class="max-w-[65ch] text-sm leading-relaxed text-muted-foreground">
        {readiness.initialSetup
          ? "Choose a connection and model, then check that you’re ready to record."
          : "Review what needs attention below. Other tasks are still available from the tabs above."}
      </p>
    </header>

    <div
      class:initial={readiness.initialSetup}
      class="setup-columns grid min-w-0 items-start gap-4"
    >
      <div class="flex min-w-0 flex-col gap-3">
        <section
          class="overflow-hidden rounded-xl border border-hairline bg-layer-fill"
          aria-label="Next setup step"
        >
          <div class="space-y-3 p-5">
            <h3 class="text-sm font-semibold" aria-live="polite">
              {readiness.canComplete
                ? "Ready to record"
                : readiness.initialSetup
                  ? "Next step"
                  : "Needs attention"}
            </h3>
            {#if readiness.canComplete}
              <p class="text-sm leading-relaxed text-muted-foreground">
                Your connection check passed, and your microphone and shortcut are ready.
              </p>
            {:else}
              <div class="space-y-4">
                {#each remaining as step (step.id)}
                  <div class="flex items-start gap-2.5">
                    {#if step.status === "attention"}<CircleAlertIcon
                        class="mt-0.5 size-4 shrink-0 text-warning"
                      />
                    {:else if step.id === "microphone"}<LoaderCircleIcon
                        class="mt-0.5 size-4 shrink-0 animate-spin text-muted-foreground motion-reduce:animate-none"
                      />{/if}
                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium">{step.label}</p>
                      <p class="mt-1 text-xs leading-relaxed break-words text-muted-foreground">
                        {step.detail}
                      </p>
                    </div>
                    {#if step.settingsSection && !(action.kind === "settings" && action.step === step.id) && step.status === "attention"}
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        aria-label={`Review ${step.label} settings`}
                        onclick={() => onOpenSettings(step.settingsSection!)}>Review</Button
                      >
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
          <div class="flex flex-wrap items-center gap-2 border-t border-hairline px-5 py-3">
            <Button disabled={busy || action.kind === "wait"} onclick={proceed} class="min-w-0">
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
            {#if !readiness.initialSetup}<Button
                variant="ghost"
                size="sm"
                disabled={busy}
                onclick={onDismiss}>Back to workspace</Button
              >{/if}
          </div>
        </section>

        {#if complete.length}
          <details class="group rounded-xl border border-hairline bg-layer-fill">
            <summary
              class="flex cursor-pointer list-none items-center gap-2 px-4 py-3 text-sm focus-visible:outline-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden"
            >
              <CheckIcon class="size-4 text-success" /><span class="flex-1"
                >{complete.length} {complete.length === 1 ? "check ready" : "checks ready"}</span
              >
              <ChevronDownIcon class="size-4 text-muted-foreground group-open:rotate-180" />
            </summary>
            <div class="divide-y divide-hairline border-t border-hairline">
              {#each complete as step (step.id)}{@render readyStep(step)}{/each}
            </div>
          </details>
        {/if}
        {#if !readiness.initialSetup && connectionNeedsAttention && serverControls}
          <details class="group rounded-xl border border-hairline bg-layer-fill">
            <summary
              class="flex cursor-pointer list-none items-center justify-between gap-2 px-4 py-3 text-sm focus-visible:outline-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden"
            >
              Connection and model<ChevronDownIcon
                class="size-4 text-muted-foreground group-open:rotate-180"
              />
            </summary>
            <div class="border-t border-hairline p-5">{@render serverControls()}</div>
          </details>
        {/if}
      </div>
      {#if readiness.initialSetup && serverControls}
        <section
          class="setup-controls min-w-0 rounded-xl border border-hairline bg-layer-fill p-5"
          aria-label="Transcription connection"
        >
          {@render serverControls()}
        </section>
      {/if}
    </div>
  </div>
</section>

{#snippet readyStep(step: ReadinessStep)}
  <div class="flex items-start gap-3 px-4 py-3">
    <div class="min-w-0 flex-1">
      <p class="text-xs font-medium">{step.label}</p>
      {#if step.id === "shortcut"}<div class="mt-1">
          <ShortcutKeys value={step.detail} label="Toggle recording shortcut" />
        </div>
      {:else}<p class="mt-1 text-xs leading-relaxed break-words text-muted-foreground">
          {step.detail}
        </p>{/if}
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
  @container (min-width: 800px) {
    .setup-columns.initial {
      grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr);
    }
    .setup-columns.initial > .setup-controls {
      grid-column: 1;
      grid-row: 1;
    }
    .setup-columns.initial > :first-child {
      grid-column: 2;
      grid-row: 1;
    }
  }
  .setup-columns:not(.initial) {
    max-width: 40rem;
  }
</style>
