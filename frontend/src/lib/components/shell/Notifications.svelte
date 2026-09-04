<script lang="ts">
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import InfoIcon from "@lucide/svelte/icons/info";
  import XIcon from "@lucide/svelte/icons/x";

  import { orderMessages, type Message, type MessageTone } from "$lib/utils/messages";

  let { messages = [] }: { messages?: Message[] } = $props();
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

<!--
  The messages channel. It carries only what the surrounding UI cannot say for
  itself: the transport states its own progress, so nothing routine belongs
  here.

  It collapses to zero height rather than unmounting and sits between the
  transport and the columns, so a message arriving pushes the rack and the
  transcript list down together rather than resizing either of them.
-->
<div class="channel" class:open={messages.length > 0}>
  <div class="channel-inner">
    <div class="flex flex-col gap-1.5 pb-2.5">
      {#each orderedMessages as message (message.id)}
        {@const Icon = icons[message.tone]}
        <div
          class="flex items-start gap-2.5 rounded-md border px-2.5 py-2 text-[11.5px] leading-relaxed {tones[
            message.tone
          ]}"
          role={message.tone === "error" ? "alert" : "status"}
          aria-atomic="true"
        >
          <Icon class="mt-px size-[14px] shrink-0 {marks[message.tone]}" />
          <p class="min-w-0 flex-1 text-card-foreground">{message.text}</p>
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
      {/each}
    </div>
  </div>
</div>

<style>
  .channel {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    transition:
      grid-template-rows 260ms ease,
      opacity 200ms ease;
  }
  .channel.open {
    grid-template-rows: 1fr;
    opacity: 1;
  }
  .channel-inner {
    overflow: hidden;
    min-height: 0;
  }

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

  @media (prefers-reduced-motion: reduce) {
    .channel {
      transition: none;
    }
  }
</style>
