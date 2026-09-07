<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import { Switch } from "$lib/components/ui/switch";

  let {
    label,
    checked,
    disabled,
    pending = false,
    saved = false,
    onChange,
  }: {
    label: string;
    checked: boolean;
    disabled: boolean;
    pending?: boolean;
    saved?: boolean;
    onChange: (checked: boolean) => Promise<void>;
  } = $props();
  const id = $props.id();
</script>

<div class="flex min-h-9 items-center justify-between gap-3">
  <label for={id} class="cursor-pointer text-[13px] text-foreground"
    >{label}</label
  >
  <div class="flex shrink-0 items-center gap-2">
    {#if pending}<LoaderCircleIcon
        class="size-3.5 animate-spin text-muted-foreground"
        aria-hidden="true"
      />
    {:else if saved}<CheckIcon
        class="size-3.5 text-success"
        aria-hidden="true"
      />{/if}
    <Switch
      {id}
      {checked}
      {disabled}
      onCheckedChange={(value) => void onChange(value)}
      aria-label={label}
    />
  </div>
</div>
