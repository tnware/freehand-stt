<script lang="ts">
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import InfoIcon from "@lucide/svelte/icons/info";
  import XIcon from "@lucide/svelte/icons/x";

  import { orderMessages, type Message, type MessageTone } from "$lib/utils/messages";

  let { messages = [], abovePlayback = false }: { messages?: Message[]; abovePlayback?: boolean } =
    $props();
  const orderedMessages = $derived(orderMessages(messages));

  const icons = {
    info: InfoIcon,
    error: CircleAlertIcon,
    success: CircleCheckIcon,
  };

  const tones: Record<MessageTone, string> = {
    info: "border-accent-edge bg-accent-wash",
    error: "border-destructive/22 bg-destructive/8",
    success: "border-success/22 bg-success/8",
  };

  const marks: Record<MessageTone, string> = {
    info: "text-accent-text",
    error: "text-destructive",
    success: "text-success",
  };
</script>

<!-- Shared notices float above the workspace footer; task errors stay with their controls. -->
<aside
  aria-label="Notifications"
  class="pointer-events-none fixed z-40 {abovePlayback
    ? 'inset-x-4 bottom-52'
    : 'right-4 bottom-20 w-[420px] max-w-[calc(100vw-32px)]'}"
>
  <div
    class="pointer-events-auto max-h-[min(40vh,24rem)] space-y-2 overflow-y-auto overscroll-contain rounded-md"
  >
    {#each orderedMessages as message (message.id)}
      {@const Icon = icons[message.tone]}
      <div class="rounded-md bg-popover shadow-lg">
        <div
          class="flex items-start gap-2.5 rounded-md border px-3 py-2.5 text-xs leading-relaxed {tones[
            message.tone
          ]}"
          role={message.tone === "error" ? "alert" : "status"}
          aria-atomic="true"
        >
          <Icon class="mt-px size-[14px] shrink-0 {marks[message.tone]}" />
          <p
            class="min-w-0 flex-1 break-words text-card-foreground"
            class:line-clamp-2={message.tone === "error"}
          >
            {message.text}
          </p>
          {#if message.tone === "error"}<FeedbackDetails
              title="Action could not be completed"
              label="Error details"
              message={message.text}
            />{/if}
          {#if message.onDismiss}
            <button
              type="button"
              class="dismiss"
              aria-label="Dismiss this message"
              onclick={message.onDismiss}
            >
              <XIcon class="size-[13px]" />
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</aside>

<style>
  .dismiss {
    display: grid;
    place-items: center;
    width: 1.375rem;
    height: 1.375rem;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    color: var(--muted-foreground);
    transition: background-color 120ms ease;
  }
  .dismiss:hover {
    background-color: var(--hairline);
  }
  .dismiss:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
</style>
