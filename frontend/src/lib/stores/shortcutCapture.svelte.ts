import {
  ShortcutCaptureOutcome,
  type ShortcutCaptureProgress,
} from "$bindings/input";
import * as Service from "$bindings/input/service";
import type { ShortcutPolicy } from "$bindings/hotkey";
import type { Settings } from "$lib/state";
import {
  ShortcutAction,
  setShortcutValue,
  shortcutAssignments,
} from "$lib/utils/shortcuts";

export type ShortcutFeedback = {
  state: "captured" | "saved" | "cancelled" | "error";
  message: string;
};

const emptyFeedback = (): Partial<Record<ShortcutAction, ShortcutFeedback | null>> => ({
  [ShortcutAction.ToggleRecording]: null,
  [ShortcutAction.ShowFreehand]: null,
  [ShortcutAction.HoldToTalk]: null,
});

type ShortcutService = Pick<
  typeof Service,
  "ShortcutPolicies" | "CaptureShortcut" | "CancelShortcutCapture"
>;

/** Drives native shortcut policy discovery, chord capture, and row feedback. */
export class ShortcutCapture {
  active = $state<ShortcutAction | null>(null);
  preview = $state("");
  policies = $state<ShortcutPolicy[]>([]);
  policyError = $state("");
  feedback = $state<Partial<Record<ShortcutAction, ShortcutFeedback | null>>>(emptyFeedback());

  readonly #service: ShortcutService;
  #policyRequest: Promise<void> | null = null;
  #captureGeneration = 0;

  constructor(service: ShortcutService = Service) {
    this.#service = service;
  }

  get capturing(): boolean {
    return this.active !== null;
  }

  get policiesLoading(): boolean {
    return this.policies.length === 0 && this.policyError === "";
  }

  policyFor(action: ShortcutAction): ShortcutPolicy | undefined {
    return this.policies.find((policy) => policy.action === action);
  }

  feedbackFor(action: ShortcutAction): ShortcutFeedback | null {
    return this.feedback[action] ?? null;
  }

  /** True while another shortcut is being recorded. */
  busyElsewhere(action: ShortcutAction): boolean {
    return this.capturing && this.active !== action;
  }

  async loadPolicies() {
    if (this.policies.length > 0) return;
    if (this.#policyRequest) return this.#policyRequest;
    this.policyError = "";
    this.#policyRequest = this.#service
      .ShortcutPolicies()
      .then((policies) => {
        this.policies = policies ?? [];
        if (this.policies.length === 0) {
          this.policyError = "Shortcut policy is unavailable.";
        }
      })
      .catch((cause) => {
        this.policyError = String(cause).replace(/^Error:\s*/, "");
      })
      .finally(() => {
        this.#policyRequest = null;
      });
    return this.#policyRequest;
  }

  applyProgress(progress: ShortcutCaptureProgress) {
    if (this.active !== progress.action) return;
    this.preview = progress.shortcut;
  }

  async record(settings: Settings, action: ShortcutAction) {
    if (this.capturing) return;
    const generation = ++this.#captureGeneration;
    this.feedback[action] = null;
    this.preview = "";
    this.active = action;
    try {
      const result = await this.#service.CaptureShortcut({
        action,
        assignments: shortcutAssignments(settings),
      });
      if (generation !== this.#captureGeneration) return;
      if (result.outcome === ShortcutCaptureOutcome.ShortcutCaptured) {
        if (!result.shortcut) throw new Error("Native shortcut capture returned no chord.");
        setShortcutValue(settings, action, result.shortcut);
        this.feedback[action] = {
          state: "captured",
          message: result.message || "Captured. Save changes to apply this shortcut.",
        };
      } else if (result.outcome === ShortcutCaptureOutcome.ShortcutCancelled) {
        this.feedback[action] = {
          state: "cancelled",
          message: result.message || "Capture cancelled. The current shortcut was kept.",
        };
      } else {
        this.feedback[action] = {
          state: "error",
          message: result.message || "That shortcut could not be captured.",
        };
      }
    } catch (cause) {
      if (generation !== this.#captureGeneration) return;
      this.feedback[action] = {
        state: "error",
        message: String(cause).replace(/^Error:\s*/, ""),
      };
    } finally {
      if (generation === this.#captureGeneration) {
        if (this.active === action) this.active = null;
        this.preview = "";
      }
    }
  }

  async cancel() {
    await this.#service.CancelShortcutCapture();
  }

  clear(settings: Settings, action: ShortcutAction) {
    setShortcutValue(settings, action, "");
    this.feedback[action] = {
      state: "captured",
      message: "Cleared. Save changes to disable hold to talk.",
    };
  }

  restore(settings: Settings, policy: ShortcutPolicy) {
    setShortcutValue(settings, policy.action, policy.defaultShortcut ?? "");
    this.feedback[policy.action] = {
      state: "captured",
      message: "Recommended shortcut restored. Save changes to apply it.",
    };
  }

  markSaved() {
    for (const { action } of this.policies) {
      if (this.feedback[action]?.state === "captured") {
        this.feedback[action] = { state: "saved", message: "Saved and active." };
      }
    }
  }

  reset() {
    this.#captureGeneration++;
    this.active = null;
    this.preview = "";
    this.feedback = emptyFeedback();
  }
}

export const shortcutCapture = new ShortcutCapture();
