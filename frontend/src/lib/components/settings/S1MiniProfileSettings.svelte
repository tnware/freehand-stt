<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import * as Card from "$lib/components/ui/card";
  import * as Field from "$lib/components/ui/field";
  import { Separator } from "$lib/components/ui/separator";
  import { Textarea } from "$lib/components/ui/textarea";
  import S1MiniControls from "$lib/components/settings/S1MiniControls.svelte";
  import type { ProfileDescriptor, Settings } from "$lib/state";
  import { s1MiniControlLine } from "$lib/utils/processingProfiles";

  type ProcessorSettings = Settings["postProcessing"];
  type S1MiniPatch = Partial<
    Pick<ProcessorSettings, "styling" | "structure" | "context">
  >;

  let {
    processor,
    profile,
    disabled = false,
    onChange,
  }: {
    processor: ProcessorSettings;
    profile: ProfileDescriptor;
    disabled?: boolean;
    onChange: (patch: S1MiniPatch) => void | Promise<void>;
  } = $props();

  const controlLine = $derived(s1MiniControlLine(processor));
</script>

<Card.Root size="sm">
  <Card.Header>
    <Card.Title>S1-mini request profile</Card.Title>
    <Card.Description>
      Use S1-mini's built-in instruction with the supported style, structure,
      and context choices.
    </Card.Description>
    <Card.Action>
      <div class="flex flex-wrap justify-end gap-1.5">
        <Badge variant="secondary">English only</Badge>
        <Badge variant="secondary">Reasoning off</Badge>
      </div>
    </Card.Action>
  </Card.Header>
  <Card.Content class="flex flex-col gap-4">
    <S1MiniControls
      {processor}
      {profile}
      idPrefix="settings-s1-mini"
      {disabled}
      {onChange}
    />

    <p class="text-xs leading-relaxed text-muted-foreground">
      To write your own custom instruction, choose Generic in the Model profile
      menu.
    </p>

    <details class="text-xs text-muted-foreground">
      <summary class="cursor-pointer font-medium text-foreground"
        >S1-mini request details</summary
      >
      <Separator class="my-4" />
      <Field.Group class="gap-4">
        <Field.Field data-disabled={disabled}>
          <div class="flex items-center justify-between gap-3">
            <Field.Label for="s1-mini-system-instruction"
              >Built-in system instruction</Field.Label
            >
            <Badge variant="outline">Read only</Badge>
          </div>
          <Textarea
            id="s1-mini-system-instruction"
            value={profile.systemInstruction ?? ""}
            rows={5}
            class="min-h-28 resize-none"
            readonly
            aria-readonly="true"
          />
          <Field.Description>
            This instruction is built in and cannot be edited with the S1-mini
            profile.
          </Field.Description>
        </Field.Field>

        <Field.Field>
          <Field.Label for="s1-mini-control-line"
            >Effective control line</Field.Label
          >
          <code
            id="s1-mini-control-line"
            class="block overflow-x-auto rounded-md border border-input bg-muted px-3 py-2 text-xs text-foreground"
            >{controlLine}</code
          >
          <Field.Description>
            Freehand places this line before the raw transcript in the user
            message.
          </Field.Description>
        </Field.Field>
      </Field.Group>
    </details>
  </Card.Content>
</Card.Root>
