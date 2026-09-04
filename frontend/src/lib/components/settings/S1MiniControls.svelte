<script lang="ts">
  import * as Field from "$lib/components/ui/field";
  import * as ToggleGroup from "$lib/components/ui/toggle-group";
  import type { ProfileDescriptor, Settings } from "$lib/state";
  import { cn } from "$lib/utils";

  type ProcessorSettings = Settings["postProcessing"];
  type S1MiniPatch = Partial<
    Pick<ProcessorSettings, "styling" | "structure" | "context">
  >;

  let {
    processor,
    profile,
    idPrefix,
    disabled = false,
    compact = false,
    onChange,
  }: {
    processor: ProcessorSettings;
    profile: ProfileDescriptor;
    idPrefix: string;
    disabled?: boolean;
    compact?: boolean;
    onChange: (patch: S1MiniPatch) => void | Promise<unknown>;
  } = $props();

  const stylingOptions = $derived(profile.controls?.styling ?? []);
  const structureOptions = $derived(profile.controls?.structure ?? []);
  const contextOptions = $derived(profile.controls?.context ?? []);
  let stylingValue = $state("");
  let structureValue = $state("");
  let contextValue = $state("");

  $effect(() => {
    stylingValue = processor.styling;
    structureValue = processor.structure;
    contextValue = processor.context;
  });

  async function commitStyling(value: string) {
    if (!stylingOptions.includes(value)) {
      stylingValue = processor.styling;
      return;
    }
    if (value === processor.styling) return;
    if ((await onChange({ styling: value })) === false) stylingValue = processor.styling;
  }

  async function commitStructure(value: string) {
    if (!structureOptions.includes(value)) {
      structureValue = processor.structure;
      return;
    }
    if (value === processor.structure) return;
    if ((await onChange({ structure: value })) === false) structureValue = processor.structure;
  }

  async function commitContext(value: string) {
    if (!contextOptions.includes(value)) {
      contextValue = processor.context;
      return;
    }
    if (value === processor.context) return;
    if ((await onChange({ context: value })) === false) contextValue = processor.context;
  }

  function optionLabel(value: string): string {
    return value
      .replaceAll("-", " ")
      .replace(/\b\w/g, (character) => character.toUpperCase());
  }
</script>

<Field.Group class={cn("gap-4", compact && "gap-3")}>
  <Field.Set class="gap-2" {disabled}>
    <Field.Legend
      id={`${idPrefix}-styling-label`}
      class={cn("mb-0", compact && "caption")}
      variant="label"
    >
      Styling
      <span
        class={cn(
          "font-normal text-muted-foreground",
          compact && "font-sans text-[11px] tracking-normal normal-case",
        )}
      >
        · {optionLabel(processor.styling)}
      </span>
    </Field.Legend>
    <ToggleGroup.Root
      id={`${idPrefix}-styling`}
      class="mt-1 grid w-full grid-cols-4"
      type="single"
      size="sm"
      bind:value={stylingValue}
      onValueChange={commitStyling}
      aria-labelledby={`${idPrefix}-styling-label`}
      disabled={disabled || stylingOptions.length < 2}
    >
      {#each stylingOptions as styling (styling)}
        <ToggleGroup.Item
          value={styling}
          aria-label={`Use ${optionLabel(styling)} styling`}
          class={cn(
            "relative isolate h-9 min-w-0 rounded-none bg-transparent px-1 pt-4 pb-0 text-[10.5px] font-normal text-muted-foreground shadow-none",
            "hover:bg-transparent hover:text-foreground data-[state=on]:bg-transparent data-[state=on]:font-semibold data-[state=on]:text-primary",
            "before:absolute before:top-1 before:left-1/2 before:z-10 before:size-2 before:-translate-x-1/2 before:rounded-full before:bg-border",
            "after:absolute after:top-[7px] after:right-1/2 after:h-0.5 after:w-full after:bg-border first:after:hidden",
            "data-[state=on]:before:bg-primary data-[state=on]:before:ring-[3px] data-[state=on]:before:ring-primary/10",
          )}
        >
          {optionLabel(styling)}
        </ToggleGroup.Item>
      {/each}
    </ToggleGroup.Root>
  </Field.Set>

  <div class={cn("grid grid-cols-2 gap-4", compact && "gap-3")}>
    <Field.Set class="gap-2" {disabled}>
      <Field.Legend
        id={`${idPrefix}-structure-label`}
        class={cn("mb-0", compact && "caption")}
        variant="label">Structure</Field.Legend>
      <ToggleGroup.Root
        class="mt-1 flex w-full flex-wrap justify-start"
        type="single"
        variant="outline"
        size="sm"
        spacing={2}
        bind:value={structureValue}
        onValueChange={commitStructure}
        aria-labelledby={`${idPrefix}-structure-label`}
        {disabled}
      >
        {#each structureOptions as option (option)}
          <ToggleGroup.Item
            value={option}
            class={cn(
              "min-w-0 font-normal text-muted-foreground data-[state=on]:border-accent-edge data-[state=on]:bg-accent-wash data-[state=on]:font-medium data-[state=on]:text-accent-text",
              compact
                ? "h-[26px] rounded-md px-2.5 text-[11px]"
                : "h-7 rounded-full px-3 text-[11.5px]",
            )}
          >
            {optionLabel(option)}
          </ToggleGroup.Item>
        {/each}
      </ToggleGroup.Root>
    </Field.Set>

    <Field.Set class="gap-2" {disabled}>
      <Field.Legend
        id={`${idPrefix}-context-label`}
        class={cn("mb-0", compact && "caption")}
        variant="label">Context</Field.Legend>
      <ToggleGroup.Root
        class="mt-1 flex w-full flex-wrap justify-start"
        type="single"
        variant="outline"
        size="sm"
        spacing={2}
        bind:value={contextValue}
        onValueChange={commitContext}
        aria-labelledby={`${idPrefix}-context-label`}
        {disabled}
      >
        {#each contextOptions as option (option)}
          <ToggleGroup.Item
            value={option}
            class={cn(
              "min-w-0 font-normal text-muted-foreground data-[state=on]:border-accent-edge data-[state=on]:bg-accent-wash data-[state=on]:font-medium data-[state=on]:text-accent-text",
              compact
                ? "h-[26px] rounded-md px-2.5 text-[11px]"
                : "h-7 rounded-full px-3 text-[11.5px]",
            )}
          >
            {optionLabel(option)}
          </ToggleGroup.Item>
        {/each}
      </ToggleGroup.Root>
    </Field.Set>
  </div>
</Field.Group>
