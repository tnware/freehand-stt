<script lang="ts">
  import type { Component, Snippet } from "svelte";
  import * as Dialog from "$lib/components/ui/dialog";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";

  let {
    open,
    title,
    description,
    icon: Icon,
    tone = "accent",
    busy = false,
    dismissible = true,
    error = "",
    errorClass = "",
    ondismiss = () => {},
    children,
    actions,
  }: {
    open: boolean;
    title: string;
    description: string;
    icon: Component;
    tone?: "accent" | "danger";
    busy?: boolean;
    dismissible?: boolean;
    error?: string;
    errorClass?: string;
    ondismiss?: () => void;
    children?: Snippet;
    actions: Snippet;
  } = $props();
  const layout = getWorkbenchLayout();
  let content = $state<HTMLElement | null>(null);
  const canDismiss = $derived(dismissible && !busy);

  $effect(() => {
    if (open) return layout?.claimNotifications("modal");
  });

  function focusInitialAction(event: Event) {
    const target = content?.querySelector<HTMLElement>(
      "[data-dialog-initial-focus]:not([disabled])",
    );
    if (target) {
      event.preventDefault();
      target.focus();
    }
  }
</script>

<Dialog.Root
  {open}
  onOpenChange={(nextOpen) => {
    if (!nextOpen && canDismiss) ondismiss();
  }}
>
  <Dialog.Content
    bind:ref={content}
    showCloseButton={canDismiss}
    escapeKeydownBehavior={canDismiss ? "close" : "ignore"}
    interactOutsideBehavior={canDismiss ? "close" : "ignore"}
    onOpenAutoFocus={focusInitialAction}
    aria-busy={busy}
    class="flex max-h-[calc(100dvh-2rem)] flex-col gap-0 overflow-hidden bg-dialog-surface p-0 shadow-float ring-dialog-stroke sm:max-w-[460px]"
  >
    <div class="min-h-0 overflow-y-auto p-5">
      <Dialog.Header>
        <div class="flex items-start gap-3">
          <div class="dialog-icon" class:danger={tone === "danger"}>
            <Icon class="size-5" aria-hidden="true" />
          </div>
          <div class="min-w-0 flex-1" class:pr-7={canDismiss}>
            <Dialog.Title
              class="break-words text-base leading-snug font-semibold"
              >{title}</Dialog.Title
            >
            <Dialog.Description class="mt-2 [overflow-wrap:anywhere]">
              {description}
            </Dialog.Description>
          </div>
        </div>
      </Dialog.Header>
      {#if children}<div class="mt-4 flex flex-col gap-3">
          {@render children()}
        </div>{/if}
      {#if error}<p
          role="alert"
          class="mt-4 rounded-md border border-destructive/25 bg-destructive/5 p-3 text-sm text-destructive [overflow-wrap:anywhere] {errorClass}"
        >
          {error}
        </p>{/if}
    </div>
    <Dialog.Footer
      class="shrink-0 flex-col border-t border-hairline bg-layer-fill p-4 sm:flex-row"
    >
      {@render actions()}
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<style>
  .dialog-icon {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    border-radius: var(--radius-md);
    color: var(--primary);
    background: color-mix(in srgb, var(--primary) 12%, transparent);
  }
  .dialog-icon.danger {
    color: var(--destructive);
    background: color-mix(in srgb, var(--destructive) 12%, transparent);
  }
</style>
