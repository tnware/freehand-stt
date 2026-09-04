/**
 * The messages channel carries only what the surrounding UI cannot say for
 * itself. The transport states its own progress and the status strip states the
 * connection, so a message here is either an action that failed, an action
 * that succeeded quietly, or an explanation of a state that looks wrong but
 * is not.
 */

export type MessageTone = "info" | "error" | "success";
export type MessageSource = "system" | "action";

export type Message = {
  /** Stable across renders, so an unchanged message does not re-animate. */
  id: string;
  tone: MessageTone;
  /** System state is always presented before outcomes of the user's actions. */
  source?: MessageSource;
  text: string;
  /** Omitted for messages the state owns: they clear when the state does. */
  onDismiss?: () => void;
};

export const orderMessages = (messages: Message[]): Message[] =>
  messages
    .map((message, index) => ({ message, index }))
    .sort((left, right) => {
      const leftOrder = left.message.source === "system" ? 0 : 1;
      const rightOrder = right.message.source === "system" ? 0 : 1;
      return leftOrder - rightOrder || left.index - right.index;
    })
    .map(({ message }) => message);
