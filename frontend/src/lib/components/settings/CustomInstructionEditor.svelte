<script lang="ts">
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Field from "$lib/components/ui/field";
  import { Textarea } from "$lib/components/ui/textarea";
  import {
    instructionBytes,
    instructionError,
  } from "$lib/utils/processingProfiles";

  let {
    value = $bindable(),
    recommended,
    maximumBytes,
    disabled = false,
  }: {
    value: string;
    recommended: string;
    maximumBytes: number;
    disabled?: boolean;
  } = $props();

  const bytes = $derived(instructionBytes(value));
  const error = $derived(instructionError(value, maximumBytes));
  const canRestore = $derived(Boolean(recommended) && value !== recommended);
</script>

<Card.Root size="sm" class="border border-card-stroke">
  <Card.Header>
    <Card.Title>Custom system instruction</Card.Title>
    <Card.Description>
      This is sent as the system message. The raw transcript is sent separately as the user
      message, so the instruction does not need a transcript placeholder. It is stored locally
      with your ordinary Freehand settings; credentials remain separate.
    </Card.Description>
    <Card.Action><Badge variant="secondary">Editable</Badge></Card.Action>
  </Card.Header>
  <Card.Content>
    <Field.Field data-invalid={Boolean(error)} data-disabled={disabled}>
      <Field.Label for="post-processing-custom-instruction">Instruction</Field.Label>
      <Textarea
        id="post-processing-custom-instruction"
        bind:value
        rows={7}
        class="min-h-36 max-h-72 resize-y"
        maxlength={maximumBytes}
        aria-invalid={Boolean(error)}
        spellcheck
        {disabled}
      />
      {#if error}
        <Field.Error>{error}</Field.Error>
      {:else}
        <Field.Description>
          Be explicit about preserving meaning and returning only the replacement transcript.
        </Field.Description>
      {/if}
    </Field.Field>
  </Card.Content>
  <Card.Footer class="justify-between gap-3 border-t">
    <span class="text-xs tabular-nums text-muted-foreground" aria-live="polite">
      {bytes.toLocaleString()} / {maximumBytes.toLocaleString()} UTF-8 bytes
    </span>
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={disabled || !canRestore}
      onclick={() => (value = recommended)}
    >
      <RotateCcwIcon data-icon="inline-start" />
      Restore recommended
    </Button>
  </Card.Footer>
</Card.Root>
