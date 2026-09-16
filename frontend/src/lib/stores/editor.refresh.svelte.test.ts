import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import type { Settings } from "$lib/state";
import {
  createEditor,
  idle,
  processingProfiles,
  serviceWithStatus,
  settings,
} from "./session-fixtures";

const changed = (): Settings => ({
  ...structuredClone(settings),
  fileTranscriptionTimeoutSeconds: 90,
});

describe("SettingsEditor refresh reconciliation", () => {
  it("keeps an event received during initialization and still loads profile metadata", async () => {
    const pending = CancellablePromise.withResolvers<Settings>();
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { GetSettings: () => pending.promise },
      }),
    );
    try {
      const loading = editor.load();
      editor.applySettingsSnapshot(changed());
      pending.resolve(settings);
      expect(await loading).toBe(true);
      expect(editor.applied).toMatchObject(changed());
      expect(editor.processingProfiles).toEqual(processingProfiles);
    } finally {
      editor.dispose();
      messages.dispose();
    }
  });

  it.each(["older-first", "newer-first"])(
    "keeps the latest requested refresh when responses finish %s",
    async (order) => {
      const older = CancellablePromise.withResolvers<Settings>();
      const newer = CancellablePromise.withResolvers<Settings>();
      const GetSettings = vi
        .fn()
        .mockImplementationOnce(() => older.promise)
        .mockImplementationOnce(() => newer.promise);
      const { editor, messages } = createEditor(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          settings: { GetSettings },
        }),
      );
      try {
        editor.applySettingsSnapshot(settings);
        const first = editor.refresh();
        const second = editor.refresh();
        if (order === "older-first") {
          older.resolve(settings);
          await first;
          newer.resolve(changed());
          await second;
        } else {
          newer.resolve(changed());
          await second;
          older.resolve(settings);
          await first;
        }
        expect(editor.applied).toMatchObject(changed());
      } finally {
        editor.dispose();
        messages.dispose();
      }
    },
  );

  it("keeps a completed save instead of an older pending read", async () => {
    const pending = CancellablePromise.withResolvers<Settings>();
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: () => pending.promise,
          SaveSettings: () => CancellablePromise.resolve(changed()),
        },
      }),
    );
    try {
      editor.applySettingsSnapshot(settings);
      const reading = editor.refresh();
      editor.draft!.fileTranscriptionTimeoutSeconds = 90;
      expect(await editor.save()).toBe(true);
      pending.resolve(settings);
      await reading;
      expect(editor.applied).toMatchObject(changed());
      expect(editor.dirty).toBe(false);
    } finally {
      editor.dispose();
      messages.dispose();
    }
  });

  it("preserves a connection credential draft and applies deferred settings after discard", async () => {
    const pending = CancellablePromise.withResolvers<Settings>();
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { GetSettings: () => pending.promise },
      }),
    );
    try {
      editor.applySettingsSnapshot(settings);
      const reading = editor.refresh();
      editor.beginConnection();
      editor.connectionDraft!.name = "Unfinished endpoint";
      editor.connectionDraft!.credentialDraft = "transient-fixture-key";
      pending.resolve(changed());
      expect(await reading).toBe(true);
      expect(editor.connectionDraft!.credentialDraft).toBe(
        "transient-fixture-key",
      );
      expect(editor.applied).toMatchObject(settings);
      expect(editor.dirty).toBe(true);
      editor.cancelConnectionEdit();
      expect(editor.connectionDraft).toBeNull();
      expect(editor.applied).toMatchObject(changed());
      expect(editor.dirty).toBe(false);
    } finally {
      editor.dispose();
      messages.dispose();
    }
  });

  it("does not erase a draft while its save is pending and subsequently fails", async () => {
    const pending = CancellablePromise.withResolvers<Settings>();
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: () => CancellablePromise.resolve(changed()),
          SaveSettings: () => pending.promise,
        },
      }),
    );
    try {
      editor.applySettingsSnapshot(settings);
      editor.draft!.fileTranscriptionTimeoutSeconds = 75;
      const saving = editor.save();
      await editor.refresh();
      pending.reject(new Error("fixture save failure"));
      expect(await saving).toBe(false);
      expect(editor.draft!.fileTranscriptionTimeoutSeconds).toBe(75);
      expect(editor.dirty).toBe(true);
      editor.discardSettingsDraft();
      expect(editor.applied).toMatchObject(changed());
    } finally {
      editor.dispose();
      messages.dispose();
    }
  });

  it("does not publish an obsolete read failure after a newer event", async () => {
    const pending = CancellablePromise.withResolvers<Settings>();
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { GetSettings: () => pending.promise },
      }),
    );
    try {
      const reading = editor.refresh();
      editor.applySettingsSnapshot(changed());
      pending.reject(new Error("obsolete read failure"));
      await reading;
      expect(editor.applied).toMatchObject(changed());
      expect(messages.error).toBe("");
    } finally {
      editor.dispose();
      messages.dispose();
    }
  });

  it.each(["success", "failure"])(
    "ignores late initialization %s after disposal and clears credentials",
    async (outcome) => {
      const pending = CancellablePromise.withResolvers<Settings>();
      const profiles =
        CancellablePromise.withResolvers<typeof processingProfiles>();
      const GetSettings = vi.fn(() => pending.promise);
      const { editor, messages } = createEditor(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          settings: {
            GetSettings,
            GetPostProcessingProfiles: () => profiles.promise,
          },
        }),
      );
      const loading = editor.load();
      editor.applySettingsSnapshot(settings);
      editor.beginConnection();
      editor.connectionDraft!.credentialDraft = "transient-fixture-key";
      editor.dispose();
      messages.dispose();
      if (outcome === "success") {
        pending.resolve(changed());
        profiles.resolve(processingProfiles);
      } else {
        pending.reject(new Error("late settings failure"));
        profiles.reject(new Error("late profile failure"));
      }
      expect(await loading).toBe(false);
      expect(editor.applied).toMatchObject(settings);
      expect(editor.connectionDraft).toBeNull();
      expect(editor.processingProfiles).toEqual([]);
      expect(messages.error).toBe("");
      expect(await editor.load()).toBe(false);
      expect(await editor.refresh()).toBe(false);
      expect(GetSettings).toHaveBeenCalledOnce();
    },
  );

  it("does not continue queued quick saves or recreate notices after disposal", async () => {
    vi.useFakeTimers();
    const pending = CancellablePromise.withResolvers<Settings>();
    const SaveSettings = vi.fn(() => pending.promise);
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    try {
      editor.applySettingsSnapshot(settings);
      const first = editor.updateQuickSettings(
        { autoInsert: false },
        "delivery",
      );
      const second = editor.updateQuickSettings(
        { historyEnabled: true },
        "history-enabled",
      );
      await Promise.resolve();
      expect(SaveSettings).toHaveBeenCalledOnce();
      editor.dispose();
      messages.dispose();
      pending.resolve(changed());
      expect(await first).toBe(false);
      expect(await second).toBe(false);
      expect(SaveSettings).toHaveBeenCalledOnce();
      expect(editor.applied).toMatchObject(settings);
      expect(editor.quickSettingsSaved).toBeNull();
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      editor.dispose();
      messages.dispose();
      vi.useRealTimers();
    }
  });
});
