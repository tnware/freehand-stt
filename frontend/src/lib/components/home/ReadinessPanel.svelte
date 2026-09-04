<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import ClockIcon from "@lucide/svelte/icons/clock";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import ArrowRightIcon from "@lucide/svelte/icons/arrow-right";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { SettingsSectionID } from "$lib/navigation";
  import type { Readiness } from "$lib/utils/readiness";
  import { cn } from "$lib/utils";

  let {
    readiness,
    testing = false,
    completing = false,
    onTestConnection,
    onComplete,
    onDismiss,
    onOpenSettings,
  }: {
    readiness: Readiness;
    testing?: boolean;
    completing?: boolean;
    onTestConnection: () => void;
    onComplete: () => void;
    onDismiss: () => void;
    onOpenSettings: (section: SettingsSectionID) => void;
  } = $props();

  const total = $derived(readiness.steps.length);
</script>

<!--
  First run and recovery are an exclusive shell state: there is nothing to
  dictate into yet, so the rack and the feed are replaced rather than stacked
  behind a banner. Only the outstanding step carries the accent.
-->
<div class="flex min-h-0 flex-1 items-center justify-center overflow-y-auto py-2">
  <div class="flex w-full max-w-[720px] flex-col gap-5">
    <div class="flex flex-col gap-2">
      <span class="caption">{readiness.initialSetup ? "First run" : "Needs attention"}</span>
      <h2 class="text-[26px] leading-tight font-semibold tracking-[-0.015em]">
        {readiness.initialSetup
          ? `${total} checks and you can dictate anywhere.`
          : "A requirement that was working is unavailable."}
      </h2>
      <p class="max-w-[60ch] text-[13.5px] leading-relaxed text-secondary-foreground">
        {readiness.initialSetup
          ? "Choose a speech service you run or trust. Test connection reads metadata without sending audio or running a model. After setup, Freehand checks the saved speech connection automatically. Update checks can be disabled in Settings → General."
          : "Your settings and transcripts are unchanged. Fix the step below, or continue and Freehand will keep working with what is still available."}
      </p>
    </div>

    <div class="overflow-hidden rounded-lg border border-card-stroke bg-card shadow-lift">
      {#each readiness.steps as step (step.id)}
        <div
          class={cn(
            "flex items-center gap-3.5 border-t border-hairline px-4 py-3 first:border-t-0",
            step.status === "attention" && "bg-accent-wash shadow-[inset_2px_0_0_var(--primary)]",
          )}
        >
          <span
            class={cn(
              "grid size-6 shrink-0 place-items-center rounded-full",
              step.status === "complete" && "bg-success/12 text-success",
              step.status === "pending" && "bg-muted text-muted-foreground",
              step.status === "attention" && "border border-accent-edge bg-accent-wash text-accent-text",
            )}
            aria-hidden="true"
          >
            {#if step.status === "complete"}
              <CheckIcon class="size-[13px]" />
            {:else if step.status === "attention"}
              <CircleAlertIcon class="size-[13px]" />
            {:else}
              <ClockIcon class="size-3" />
            {/if}
          </span>

          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <span class="text-[13.5px] font-medium">{step.label}</span>
            {#if step.id === "shortcut" && step.status === "complete"}
              <span class="mt-0.5 flex"><ShortcutKeys value={step.detail} label="Toggle recording shortcut" /></span>
            {:else}
              <span
                class={cn(
                  "text-[11.5px] leading-relaxed break-words",
                  step.id === "server" && "figure text-[10.5px]",
                  step.status === "attention" ? "text-secondary-foreground" : "text-muted-foreground",
                )}
              >
                {step.detail}
              </span>
            {/if}
          </div>

          {#if step.status === "attention" && step.settingsSection}
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={`Review ${step.label} settings`}
              onclick={() => onOpenSettings(step.settingsSection!)}
            >
              <ChevronRightIcon />
            </Button>
          {/if}
        </div>
      {/each}

      <div class="flex items-center gap-3 border-t border-hairline bg-secondary px-4 py-3.5">
        <span class="mr-auto flex items-center gap-2.5" aria-live="polite">
          <span class="flex items-center gap-1" aria-hidden="true">
            {#each readiness.steps as step (step.id)}
              <span class={cn("h-[3px] w-[18px] rounded-full", step.status === "complete" ? "bg-success" : "bg-border")}
              ></span>
            {/each}
          </span>
          <span class="figure text-[10.5px] text-muted-foreground">
            {readiness.completedCount} of {total} ready
          </span>
        </span>

        {#if readiness.initialSetup}
          <Button
            variant="outline"
            size="sm"
            disabled={!readiness.canComplete || testing || completing}
            onclick={onComplete}
          >
            {#if completing}
              <LoaderCircleIcon data-icon="inline-start" class="animate-spin motion-reduce:animate-none" />
            {/if}
            {completing ? "Finishing…" : "Finish setup"}
          </Button>
        {:else}
          <Button variant="ghost" size="sm" onclick={onDismiss}>Continue anyway</Button>
        {/if}
        <Button size="sm" disabled={!readiness.canTestConnection || testing || completing} onclick={onTestConnection}>
          {#if testing}
            <LoaderCircleIcon data-icon="inline-start" class="animate-spin motion-reduce:animate-none" />
          {:else}
            <ArrowRightIcon data-icon="inline-start" />
          {/if}
          {testing ? "Checking…" : "Test connection"}
        </Button>
      </div>
    </div>

    {#if readiness.initialSetup}
      <p class="text-[11.5px] text-muted-foreground">
        Finish setup unlocks once the check passes. You can change any of this later in Settings.
      </p>
    {/if}
  </div>
</div>
