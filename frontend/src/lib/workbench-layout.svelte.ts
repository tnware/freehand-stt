import { getContext, setContext, type Snippet } from "svelte";
import { MediaQuery, SvelteSet } from "svelte/reactivity";

export type SidebarID =
  "workflow" | "settings" | "history" | "runtimes" | "connections";
export type WorkbenchTab = "recent" | "output" | "diagnostics";
const key = Symbol("workbench-layout");
const storageKey = "freehand-workbench-layout-v1";

/** Presentation preferences only. Transcript and output data never enter storage. */
export class WorkbenchLayout {
  #modalNotificationOwners = new SvelteSet<symbol>();
  #inlineErrorOwners = new SvelteSet<symbol>();
  notificationsPaused = $derived(this.#modalNotificationOwners.size > 0);
  errorOwned = $derived(this.#inlineErrorOwners.size > 0);
  primaryOpen = $state(true);
  bottomOpen = $state(true);
  secondaryOpen = $state(true);
  compactPrimaryOpen = $state(false);
  compactSecondaryOpen = $state(false);
  tab = $state<WorkbenchTab>("recent");
  bottomSize = $state(30);
  secondarySize = $state(40);
  outputInstanceID = $state("");
  diagnosticWorkflow = $state<"voice" | "file">("voice");
  sidebars = $state<Partial<Record<SidebarID, Snippet>>>({});
  details = $state<Snippet>();
  compact = new MediaQuery("(max-width: 699px)");
  bottomAvailable = new MediaQuery("(min-height: 560px)");
  secondaryAvailable = new MediaQuery("(min-width: 1100px)");
  primaryVisible = $derived(
    this.compact.current ? this.compactPrimaryOpen : this.primaryOpen,
  );
  bottomVisible = $derived(this.bottomOpen && this.bottomAvailable.current);
  secondaryVisible = $derived(
    this.secondaryAvailable.current
      ? this.secondaryOpen
      : this.compactSecondaryOpen,
  );

  /** Presentation ownership only; releasing an owner never dismisses its message. */
  claimNotifications(scope: "modal" | "error"): () => void {
    const owners =
      scope === "modal"
        ? this.#modalNotificationOwners
        : this.#inlineErrorOwners;
    const owner = Symbol(scope);
    owners.add(owner);
    return () => {
      owners.delete(owner);
    };
  }

  togglePrimary() {
    this.compactSecondaryOpen = false;
    if (this.compact.current)
      this.compactPrimaryOpen = !this.compactPrimaryOpen;
    else this.primaryOpen = !this.primaryOpen;
  }
  toggleBottom() {
    this.bottomOpen = !this.bottomOpen;
  }
  closePrimary() {
    if (!this.compact.current) return;
    this.compactPrimaryOpen = false;
    document
      .querySelector<HTMLButtonElement>(
        'button[aria-label="Toggle primary sidebar"]',
      )
      ?.focus();
  }
  toggleSecondary() {
    this.compactPrimaryOpen = false;
    if (this.secondaryAvailable.current)
      this.secondaryOpen = !this.secondaryOpen;
    else this.compactSecondaryOpen = !this.compactSecondaryOpen;
  }
  showOutput(id: string) {
    this.outputInstanceID = id;
    this.tab = "output";
    this.bottomOpen = true;
  }
  restore() {
    try {
      const value = JSON.parse(localStorage.getItem(storageKey) ?? "null");
      if (!value || typeof value !== "object") return;
      for (const field of [
        "primaryOpen",
        "bottomOpen",
        "secondaryOpen",
      ] as const)
        if (typeof value[field] === "boolean") this[field] = value[field];
      if (["recent", "output", "diagnostics"].includes(value.tab))
        this.tab = value.tab;
      if (Number.isFinite(value.bottomSize))
        this.bottomSize = Math.max(18, Math.min(55, value.bottomSize));
      if (Number.isFinite(value.secondarySize))
        this.secondarySize = Math.max(30, Math.min(55, value.secondarySize));
    } catch {
      /* Layout storage is optional. */
    }
  }
  save() {
    const value = {
      primaryOpen: this.primaryOpen,
      bottomOpen: this.bottomOpen,
      secondaryOpen: this.secondaryOpen,
      tab: this.tab,
      bottomSize: this.bottomSize,
      secondarySize: this.secondarySize,
    };
    try {
      localStorage.setItem(storageKey, JSON.stringify(value));
    } catch {
      /* Optional. */
    }
  }
}

export function provideWorkbenchLayout(layout: WorkbenchLayout) {
  setContext(key, layout);
}
export function getWorkbenchLayout(): WorkbenchLayout | undefined {
  return getContext(key);
}
