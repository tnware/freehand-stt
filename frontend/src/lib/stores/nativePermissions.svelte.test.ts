import { describe, expect, it, vi } from "vitest";
import { NativePermissionState } from "./nativePermissions.svelte";
import type { PermissionStatus } from "$bindings/input";
import { permissionRows } from "../utils/nativePermissions";

const status: PermissionStatus = {
  required: true,
  microphone: "not-determined",
  accessibility: false,
  keyboard: false,
};
const services = () => ({
  NativePermissions: vi.fn(async () => ({ ...status })),
  RequestPermission: vi.fn(async (_kind: string) => ({
    ...status,
    microphone: "denied",
  })),
  OpenPermissionSettings: vi.fn(async (_kind: string) => {}),
});

describe("native permission recovery", () => {
  it("retains the prior snapshot and request error if the passive reread fails", async () => {
    const api = services();
    const state = new NativePermissionState(api);
    await state.refresh();
    api.RequestPermission.mockRejectedValueOnce(new Error("denied"));
    api.NativePermissions.mockRejectedValueOnce(
      new Error("reread unavailable"),
    );
    await state.request("microphone");
    expect(state.status).toEqual(status);
    expect(state.error).toBe(
      "Could not check or update permissions. Try again.",
    );
    expect(state.busy).toBe(false);
    expect(api.NativePermissions).toHaveBeenCalledTimes(2);
    expect(api.RequestPermission).toHaveBeenCalledTimes(1);
  });
  it("keeps actions suppressed during the reread and ignores it after disposal", async () => {
    const api = services();
    const state = new NativePermissionState(api);
    await state.refresh();
    let finish!: (value: PermissionStatus) => void;
    api.RequestPermission.mockRejectedValueOnce(new Error("denied"));
    api.NativePermissions.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const pending = state.request("microphone");
    await vi.waitFor(() =>
      expect(api.NativePermissions).toHaveBeenCalledTimes(2),
    );
    const error = state.error;
    expect(state.busy).toBe(true);
    await state.refresh();
    await state.request("keyboard");
    await state.openSettings("microphone");
    expect(api.NativePermissions).toHaveBeenCalledTimes(2);
    expect(api.RequestPermission).toHaveBeenCalledTimes(1);
    expect(api.OpenPermissionSettings).not.toHaveBeenCalled();
    state.dispose();
    finish({ ...status, microphone: "denied" });
    await pending;
    expect(state.status).toEqual(status);
    expect(state.error).toBe(error);
    expect(state.busy).toBe(false);
  });
  it("bounds request plus reread to one deadline and ignores a late snapshot", async () => {
    vi.useFakeTimers();
    try {
      const api = services();
      const state = new NativePermissionState(api);
      await state.refresh();
      let reject!: (reason: Error) => void;
      let finish!: (value: PermissionStatus) => void;
      api.RequestPermission.mockImplementationOnce(
        () =>
          new Promise((_, fail) => {
            reject = fail;
          }),
      );
      api.NativePermissions.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finish = resolve;
          }),
      );
      const pending = state.request("microphone");
      await vi.advanceTimersByTimeAsync(10000);
      reject(new Error("denied"));
      await vi.advanceTimersByTimeAsync(0);
      expect(api.NativePermissions).toHaveBeenCalledTimes(2);
      expect(state.busy).toBe(true);
      await vi.advanceTimersByTimeAsync(5000);
      await pending;
      expect(state.busy).toBe(false);
      expect(state.status).toEqual(status);
      expect(state.error).toContain("Try again");
      await state.refresh();
      finish({ ...status, microphone: "denied" });
      await vi.advanceTimersByTimeAsync(0);
      expect(state.status).toEqual(status);
      expect(state.error).toBe("");
      expect(api.RequestPermission).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });
  it("does not reread after settings or refresh failures", async () => {
    const api = services();
    const state = new NativePermissionState(api);
    api.OpenPermissionSettings.mockRejectedValueOnce(new Error("unavailable"));
    await state.openSettings("microphone");
    expect(api.NativePermissions).not.toHaveBeenCalled();
    api.NativePermissions.mockRejectedValueOnce(new Error("unavailable"));
    await state.refresh();
    expect(api.NativePermissions).toHaveBeenCalledTimes(1);
    expect(api.RequestPermission).not.toHaveBeenCalled();
  });
  it("bounds stalled checks and ignores late results after retry", async () => {
    vi.useFakeTimers();
    try {
      const api = services();
      let finish!: (value: PermissionStatus) => void;
      api.NativePermissions.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finish = resolve;
          }),
      );
      const state = new NativePermissionState(api);
      const pending = state.refresh();
      await vi.advanceTimersByTimeAsync(15000);
      expect(state.busy).toBe(false);
      await pending;
      expect(state.error).toContain("Try again");
      await state.refresh();
      finish({ ...status, microphone: "authorized" });
      await Promise.resolve();
      expect(state.status?.microphone).toBe("not-determined");
    } finally {
      vi.useRealTimers();
    }
  });
  it("does not launch another prompt while a timed-out request is unsettled", async () => {
    vi.useFakeTimers();
    try {
      const api = services();
      let reject!: (reason: Error) => void;
      api.RequestPermission.mockImplementationOnce(
        () =>
          new Promise((_, fail) => {
            reject = fail;
          }),
      );
      const state = new NativePermissionState(api);
      const pending = state.request("microphone");
      await vi.advanceTimersByTimeAsync(15000);
      await pending;
      expect(state.busy).toBe(false);
      await state.request("microphone");
      expect(api.RequestPermission).toHaveBeenCalledTimes(1);
      reject(new Error("late denial"));
      await vi.advanceTimersByTimeAsync(0);
      expect(api.NativePermissions).not.toHaveBeenCalled();
      expect(state.status).toBeNull();
      expect(state.error).toContain("Try again");
      await state.request("microphone");
      expect(api.RequestPermission).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });
  it("ignores pending completions after disposal and refuses new requests", async () => {
    const api = services();
    let finish!: (value: PermissionStatus) => void;
    api.NativePermissions.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const state = new NativePermissionState(api);
    const pending = state.refresh();
    state.dispose();
    finish(status);
    await pending;
    await state.request("microphone");
    expect(state.status).toBeNull();
    expect(state.busy).toBe(false);
    expect(api.RequestPermission).not.toHaveBeenCalled();
  });
  it("reports status failures and allows an explicit retry", async () => {
    const api = services();
    api.NativePermissions.mockRejectedValueOnce(new Error("unavailable"));
    const state = new NativePermissionState(api);
    await state.refresh();
    expect(state.error).toContain("Try again");
    expect(state.busy).toBe(false);
    await state.refresh();
    expect(state.error).toBe("");
    expect(state.status).toEqual(status);
  });
  it("coalesces focus refreshes while a request is pending", async () => {
    const api = services();
    let finish!: (value: PermissionStatus) => void;
    api.RequestPermission.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const state = new NativePermissionState(api);
    const requesting = state.request("accessibility");
    expect(state.busy).toBe(true);
    await state.refresh();
    expect(api.NativePermissions).not.toHaveBeenCalled();
    finish({ ...status, accessibility: true });
    await requesting;
    expect(state.status?.accessibility).toBe(true);
    expect(state.busy).toBe(false);
  });
  it("checks status without requesting access and retains a denied result", async () => {
    const api = services();
    const state = new NativePermissionState(api);
    expect(api.NativePermissions).not.toHaveBeenCalled();
    await state.refresh();
    expect(state.status?.microphone).toBe("not-determined");
    expect(api.RequestPermission).not.toHaveBeenCalled();
    api.RequestPermission.mockRejectedValueOnce(
      new Error("Microphone access denied. Open System Settings."),
    );
    api.NativePermissions.mockResolvedValueOnce({
      ...status,
      microphone: "denied",
    });
    await state.request("microphone");
    expect(api.RequestPermission).toHaveBeenCalledExactlyOnceWith("microphone");
    expect(api.NativePermissions).toHaveBeenCalledTimes(2);
    expect(state.status?.microphone).toBe("denied");
    expect(permissionRows(state.status)[0].action).toBe("settings");
    expect(state.error).toContain("Try again");
    await state.openSettings("microphone");
    expect(api.OpenPermissionSettings).toHaveBeenCalledWith("microphone");
  });
});
