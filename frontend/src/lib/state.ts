import {
  RecordingMode,
  AutoStopState,
  SegmentPhase,
  State,
  VADState,
  type Status,
} from "$bindings/dictation";
import {
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
} from "$bindings/connection";
import {
  FileTranscriptionPhase,
  type FileTranscriptionDelta,
  type FileTranscriptionStatus,
} from "$bindings/filetranscription";
import {
  HistoryOutcome,
  HistoryProcessingStatus,
  HistoryResponseMode,
  HistorySource,
  HistoryTextVersion,
  type HistoryEntry,
  type HistoryPerformanceDetails,
  type HistoryProcessingDetails,
  type HistoryResponseDetails,
  type HistoryRunDetails,
  type HistorySegmentDetails,
  type HistoryUsageDetails,
} from "$bindings/history";
import {
  AppearanceMode,
  AuthenticationMode,
  OverlayAnchor,
  OverlayLayout,
  OverlayMotion,
  OverlaySurface,
  OverlayVisibility,
  OverlayVisualizer,
  PostProcessingPreset,
  VADMode,
} from "$bindings/config";
import { Mode as InsertionMode } from "$bindings/insertion";
import type { ProfileDescriptor } from "$bindings/postprocess";
import {
  Phase as TTSPhase,
  Source as TTSSource,
  type Status as TTSStatus,
} from "$bindings/tts";

export {
  ConnectionErrorKind,
  ConnectionProbe,
  FileTranscriptionPhase,
  HistoryOutcome,
  HistoryProcessingStatus,
  HistoryResponseMode,
  HistorySource,
  HistoryTextVersion,
  InsertionMode,
  RecordingMode,
  AutoStopState,
  ModelPresence,
  SegmentPhase,
  State,
  VADState,
  VADMode,
  AppearanceMode,
  AuthenticationMode,
  OverlayAnchor,
  OverlayLayout,
  OverlayMotion,
  OverlaySurface,
  OverlayVisibility,
  OverlayVisualizer,
  PostProcessingPreset,
  TTSPhase,
  TTSSource,
};
export type {
  HistoryEntry,
  HistoryPerformanceDetails,
  HistoryProcessingDetails,
  HistoryResponseDetails,
  HistoryRunDetails,
  HistorySegmentDetails,
  HistoryUsageDetails,
  ProfileDescriptor,
  Status,
  TTSStatus,
};
export type { Device } from "$bindings/audio";
export type { ConnectionResult };
export type { SettingsDTO as Settings } from "$bindings/settings";
export type { FileTranscriptionDelta, FileTranscriptionStatus };

export const elapsedSeconds = (status: Status, now: number): number => {
  if (!status.startedAt || status.state === State.Idle) return 0;
  return Math.max(0, Math.floor((now - Date.parse(status.startedAt)) / 1000));
};
