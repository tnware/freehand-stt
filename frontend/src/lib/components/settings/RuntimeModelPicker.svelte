<script lang="ts">
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  let {
    id,
    value = $bindable(),
    models = [],
    serverLoaded = false,
    busy = false,
    onDiscover,
  }: {
    id: string;
    value: string;
    models?: string[];
    serverLoaded?: boolean;
    busy?: boolean;
    onDiscover: () => void;
  } = $props();
</script>

<ValueRow
  {id}
  label="Model"
  hint={serverLoaded
    ? "The model is loaded by the whisper.cpp server. Freehand does not load or switch it."
    : "Enter a model ID or list the models advertised by the selected connection. This choice belongs to this feature."}
>
  {#snippet control()}{#if serverLoaded}<span {id} class="text-sm text-muted-foreground"
        >Server-loaded model</span
      >{:else}<div class="flex min-w-0 flex-col gap-2">
        <ValueInput
          {id}
          bind:value
          spellcheck={false}
          placeholder="Model ID"
        />{#if models.length}<Select.Root type="single" {value} onValueChange={(v) => (value = v)}
            ><Select.Trigger class="w-full" aria-label="Choose a discovered model"
              >Choose from discovered models</Select.Trigger
            ><Select.Content
              >{#each models as model (model)}<Select.Item value={model} label={model}
                  >{model}</Select.Item
                >{/each}</Select.Content
            ></Select.Root
          >{/if}
      </div>{/if}{/snippet}
  {#snippet action()}<Button variant="secondary" size="sm" disabled={busy} onclick={onDiscover}
      >{#if busy}<LoaderCircleIcon class="animate-spin" />{/if}{serverLoaded
        ? "Check server"
        : "List models"}</Button
    >{/snippet}
</ValueRow>
