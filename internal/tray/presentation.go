// Package tray owns Freehand's native system-tray presentation and actions.
// It consumes bounded domain snapshots; transcript text and file identities
// never enter the tray model.
package tray

import (
	"fmt"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
)

type operationSource uint8

const (
	sourceNone operationSource = iota
	sourceDictation
	sourceFile
)

type snapshot struct {
	dictation  dictation.Status
	file       filetranscription.FileTranscriptionStatus
	latest     operationSource
	last       string
	mainWindow bool
}

type presentation struct {
	status      string
	detail      string
	tooltip     string
	cancel      operationSource
	cancelLabel string
	copy        operationSource
	windowLabel string
}

func present(state snapshot) presentation {
	result := presentation{
		status:      "Ready",
		detail:      state.last,
		tooltip:     "Freehand — Ready",
		windowLabel: "Show Freehand",
	}
	if state.mainWindow {
		result.windowLabel = "Hide Freehand"
	}

	if state.dictation.CanCancel {
		result.cancel = sourceDictation
		result.cancelLabel = "Cancel dictation"
		if state.dictation.State == dictation.Recording {
			result.cancelLabel = "Cancel recording"
		}
	} else if state.file.CanCancel {
		result.cancel = sourceFile
		result.cancelLabel = "Cancel audio transcription"
	}

	switch {
	case activeDictation(state.dictation.State):
		applyDictation(&result, state.dictation)
	case activeFile(state.file.Phase):
		applyFile(&result, state.file)
	case state.latest == sourceDictation && state.dictation.State == dictation.Failed:
		applyDictation(&result, state.dictation)
	case state.latest == sourceFile && state.file.Phase == filetranscription.FileTranscriptionFailed:
		applyFile(&result, state.file)
	case state.latest == sourceFile && state.file.Phase == filetranscription.FileTranscriptionCompleted:
		applyFile(&result, state.file)
	case state.latest == sourceFile && state.file.Phase == filetranscription.FileTranscriptionSelected:
		result.status = "Audio file selected"
		result.detail = "Open Freehand to transcribe"
	}
	if !activeDictation(state.dictation.State) && !activeFile(state.file.Phase) {
		switch {
		case state.latest == sourceDictation && state.dictation.CanCopy:
			result.copy = sourceDictation
		case state.latest == sourceFile && state.file.CanCopy:
			result.copy = sourceFile
		}
	}
	if state.latest == sourceFile && state.file.Phase == filetranscription.FileTranscriptionCompleted && state.last != "" {
		result.detail = state.last
	}
	result.tooltip = boundedTooltip("Freehand — " + result.status)
	return result
}

func activeDictation(state dictation.State) bool {
	switch state {
	case dictation.Recording, dictation.Transcribing, dictation.PostProcessing, dictation.Ready, dictation.Cancelling:
		return true
	default:
		return false
	}
}

func activeFile(phase filetranscription.FileTranscriptionPhase) bool {
	switch phase {
	case filetranscription.FileTranscriptionUploading,
		filetranscription.FileTranscriptionProcessing,
		filetranscription.FileTranscriptionStreaming,
		filetranscription.FileTranscriptionCancelling:
		return true
	default:
		return false
	}
}

func applyDictation(result *presentation, status dictation.Status) {
	switch status.State {
	case dictation.Recording:
		result.status = "Recording"
		switch {
		case status.SegmentPhase != "":
			result.detail = fmt.Sprintf("Checkpoint %d · %s", max(status.SegmentNumber, 1), segmentPhaseLabel(status.SegmentPhase))
		case status.AutoStopState == dictation.AutoStopCountdown:
			result.detail = "Silence detected · automatic stop pending"
		case status.VADState == dictation.VADSpeech:
			result.detail = "Speech detected"
		case status.VADState == dictation.VADSilence:
			result.detail = "Silence detected · listening"
		default:
			result.detail = "Listening for speech"
		}
	case dictation.Transcribing:
		result.status = "Transcribing"
		result.detail = segmentDetail(status, "Processing recorded audio")
	case dictation.PostProcessing:
		result.status = "Cleaning transcript"
		result.detail = "Applying post-processing"
	case dictation.Ready:
		result.status = "Finishing dictation"
		result.detail = "Preparing transcript delivery"
	case dictation.Cancelling:
		result.status = "Cancelling dictation"
		result.detail = "Stopping current work"
	case dictation.Failed:
		if status.CanCopy {
			result.status = "Transcript needs attention"
			result.detail = "Copy the completed transcript from the tray"
			return
		}
		result.status = "Dictation failed"
		result.detail = "Open Freehand for details"
	}
}

func applyFile(result *presentation, status filetranscription.FileTranscriptionStatus) {
	switch status.Phase {
	case filetranscription.FileTranscriptionUploading:
		result.status = "Uploading audio"
		result.detail = uploadDetail(status.BytesUploaded, status.FileSize)
	case filetranscription.FileTranscriptionProcessing:
		result.status = "Transcribing audio file"
		result.detail = "Waiting for the completed transcript"
	case filetranscription.FileTranscriptionStreaming:
		result.status = "Transcribing audio file"
		result.detail = "Receiving transcript"
	case filetranscription.FileTranscriptionCancelling:
		result.status = "Cancelling audio transcription"
		result.detail = "Stopping current work"
	case filetranscription.FileTranscriptionCompleted:
		result.status = "Audio transcript ready"
		result.detail = "Completed transcript available"
	case filetranscription.FileTranscriptionFailed:
		if status.CanCopy {
			result.status = "Partial transcript available"
			result.detail = "Copy the recovered text or open Freehand"
			return
		}
		result.status = "Audio transcription failed"
		result.detail = "Open Freehand for details"
	}
}

func segmentDetail(status dictation.Status, fallback string) string {
	if status.SegmentNumber <= 0 {
		return fallback
	}
	return fmt.Sprintf("Checkpoint %d · %s", status.SegmentNumber, segmentPhaseLabel(status.SegmentPhase))
}

func segmentPhaseLabel(phase dictation.SegmentPhase) string {
	label := strings.ReplaceAll(string(phase), "-", " ")
	if label == "" {
		return "processing"
	}
	return label
}

func uploadDetail(uploaded, total int64) string {
	if total <= 0 {
		return "Sending audio to the configured endpoint"
	}
	percent := uploaded * 100 / total
	percent = min(max(percent, 0), 100)
	return fmt.Sprintf("%d%% uploaded", percent)
}

func boundedTooltip(value string) string {
	const maxRunes = 120
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes-1]) + "…"
}

func elapsedLabel(started, completed time.Time) string {
	if started.IsZero() || completed.Before(started) {
		return ""
	}
	duration := completed.Sub(started)
	if duration < time.Second {
		return "<1s"
	}
	if duration < 10*time.Second {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	}
	minutes := int(duration / time.Minute)
	seconds := int(duration/time.Second) % 60
	return fmt.Sprintf("%dm %02ds", minutes, seconds)
}

func withElapsed(label string, started, completed time.Time) string {
	elapsed := elapsedLabel(started, completed)
	if elapsed == "" {
		return label
	}
	return label + " · " + elapsed
}
