import type { DiffPart } from "./textDiff";

export interface TranscriptContent {
  key: string;
  text: string;
  partial?: string;
  parts?: DiffPart[];
}

export function hasTranscriptSelection(node: HTMLElement): boolean {
  const selection = node.ownerDocument.getSelection();
  if (!selection || selection.isCollapsed) return false;
  for (let index = 0; index < selection.rangeCount; index++) {
    if (selection.getRangeAt(index).intersectsNode(node)) return true;
  }
  return false;
}

function scrollViewport(node: HTMLElement): HTMLElement | null {
  for (let parent = node.parentElement; parent; parent = parent.parentElement) {
    if (
      /^(auto|scroll)$/.test(getComputedStyle(parent).overflowY) &&
      parent.scrollHeight > parent.clientHeight
    )
      return parent;
  }
  return null;
}

/** Plain text only. Hold a selected snapshot until selection clears, never inference state. */
export function transcriptReader(node: HTMLElement, initial: TranscriptContent) {
  const doc = node.ownerDocument;
  let latest = initial;
  let displayed: TranscriptContent | undefined;

  function render() {
    if (displayed?.key === latest.key && hasTranscriptSelection(node)) return;
    if (displayed?.key !== latest.key && hasTranscriptSelection(node)) {
      doc.getSelection()?.removeAllRanges();
    }
    if (
      displayed?.key === latest.key &&
      displayed.text === latest.text &&
      displayed.partial === latest.partial &&
      displayed.parts === latest.parts
    )
      return;

    const fragment = doc.createDocumentFragment();
    if (latest.parts) {
      for (const part of latest.parts) {
        const span = doc.createElement("span");
        if (part.kind !== "equal") span.className = `diff-${part.kind}`;
        span.textContent = part.text;
        fragment.append(span);
      }
    } else if (latest.text) fragment.append(doc.createTextNode(latest.text));
    if (latest.partial) {
      const span = doc.createElement("span");
      span.className = "text-muted-foreground";
      span.textContent = latest.partial;
      fragment.append(span);
    }
    node.replaceChildren(fragment);
    displayed = latest;
  }

  function keydown(event: KeyboardEvent) {
    if (event.isComposing || event.altKey || event.target !== node) return;
    if ((event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === "a") {
      event.preventDefault();
      const range = doc.createRange();
      range.selectNodeContents(node);
      const selection = doc.getSelection();
      selection?.removeAllRanges();
      selection?.addRange(range);
      return;
    }
    // Ctrl+C and selection-extension keys retain the native browser behavior.
    if (event.ctrlKey || event.metaKey || event.shiftKey) return;
    const viewport = scrollViewport(node);
    if (!viewport) return;
    const page = Math.max(1, viewport.clientHeight - 28);
    const destinations: Record<string, number> = {
      ArrowDown: viewport.scrollTop + 28,
      ArrowUp: viewport.scrollTop - 28,
      PageDown: viewport.scrollTop + page,
      PageUp: viewport.scrollTop - page,
      Home: 0,
      End: viewport.scrollHeight,
    };
    if (!(event.key in destinations)) return;
    event.preventDefault();
    viewport.scrollTop = destinations[event.key];
  }

  node.addEventListener("keydown", keydown);
  doc.addEventListener("selectionchange", render);
  render();
  return {
    update(next: TranscriptContent) {
      latest = next;
      render();
    },
    destroy() {
      node.removeEventListener("keydown", keydown);
      doc.removeEventListener("selectionchange", render);
    },
  };
}
