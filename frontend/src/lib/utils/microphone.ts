import type { Device } from "$lib/state";

/**
 * Sentinel for "follow the Windows default device". It is deliberately not the
 * empty string, because an empty select value is indistinguishable from an
 * unset one, while the backend stores the empty string for this case.
 */
export const SYSTEM_DEFAULT_MICROPHONE = "__system_default_microphone__";

export const SYSTEM_DEFAULT_LABEL = "System default microphone";

/** Converts a stored setting into the value the select binds to. */
export const microphoneChoiceFor = (microphoneID: string | undefined): string =>
  microphoneID || SYSTEM_DEFAULT_MICROPHONE;

/** Converts the select value back into the value the backend stores. */
export const microphoneIDFor = (choice: string): string =>
  choice === SYSTEM_DEFAULT_MICROPHONE ? "" : choice;

export const microphoneLabel = (choice: string, devices: Device[]): string => {
  if (choice === SYSTEM_DEFAULT_MICROPHONE) return SYSTEM_DEFAULT_LABEL;
  return devices.find((device) => device.id === choice)?.name ?? "Unavailable microphone";
};

/**
 * A configured device can disappear when it is unplugged or the driver changes
 * its identifier. That is a warning rather than an error. The saved choice is
 * kept for reconnection instead of silently switching microphones or rewriting
 * configuration.
 */
export const microphoneMissing = (choice: string, devices: Device[]): boolean =>
  choice !== SYSTEM_DEFAULT_MICROPHONE && !devices.some((device) => device.id === choice);

/** Drops the unnamed placeholder entries the capture layer can report. */
export const usableDevices = (devices: Device[] | null): Device[] =>
  (devices ?? []).filter((device) => device.id !== "" && device.name.trim() !== "");
