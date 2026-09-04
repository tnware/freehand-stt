<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Switch } from "$lib/components/ui/switch";

  let {
    id,
    title,
    description,
    checked = $bindable(false),
    disabled = false,
  }: {
    id: string;
    title: string;
    description: string;
    checked?: boolean;
    disabled?: boolean;
  } = $props();
</script>

<div class="flex items-center justify-between gap-6 py-1">
  <div class="min-w-0 space-y-1">
    <Label for={id}>{title}</Label>
    <p class="text-xs leading-5 text-muted-foreground">{description}</p>
  </div>
  <Switch {id} bind:checked {disabled} aria-label={title} />
</div>
