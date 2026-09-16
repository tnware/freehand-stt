//go:build windows

package input

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/insertion"
	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var getForegroundWindow = user32.NewProc("GetForegroundWindow")
var getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
var sendInput = user32.NewProc("SendInput")
var getAsyncKeyState = user32.NewProc("GetAsyncKeyState")

type Input struct {
	logger *slog.Logger
}

func NewInput(logger *slog.Logger) Input {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return Input{logger: logger}
}

func foreground() (insertion.Target, error) {
	h, _, e := getForegroundWindow.Call()
	if h == 0 {
		return insertion.Target{}, e
	}
	var pid uint32
	thread, _, _ := getWindowThreadProcessID.Call(h, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 || thread == 0 {
		return insertion.Target{}, errors.New("foreground process is unavailable")
	}
	info := windows.GUIThreadInfo{Size: uint32(unsafe.Sizeof(windows.GUIThreadInfo{}))}
	if e = windows.GetGUIThreadInfo(uint32(thread), &info); e != nil || info.Focus == 0 {
		return insertion.Target{}, errors.New("focused control identity is unavailable")
	}
	process, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if e != nil {
		return insertion.Target{}, errors.New("foreground process identity is unavailable")
	}
	defer windows.CloseHandle(process)
	var created, exited, kernel, user windows.Filetime
	if e = windows.GetProcessTimes(process, &created, &exited, &kernel, &user); e != nil {
		return insertion.Target{}, errors.New("foreground process creation time is unavailable")
	}
	created64 := uint64(uint32(created.HighDateTime))<<32 | uint64(created.LowDateTime)
	if created64 == 0 {
		return insertion.Target{}, errors.New("foreground process creation identity is invalid")
	}
	return insertion.Target{HWND: h, FocusHWND: uintptr(info.Focus), ThreadID: uint32(thread), ProcessID: pid, ProcessCreationTime: created64}, nil
}
func (Input) CaptureTarget() (insertion.Target, error) { return foreground() }
func (Input) Foreground() (insertion.Target, error)    { return foreground() }

type keyInput struct {
	VK        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// INPUT contains a union whose size is determined by MOUSEINPUT, even when
// the event carries KEYBDINPUT. Keep the full 32-byte native union so cbSize
// is the 40 bytes required by 64-bit Windows SendInput.
type inputEvent struct {
	Type uint32
	Pad  uint32
	Data [32]byte
}

const nativeInputSize = 40

const (
	// An ordinary transcript fits in one bounded SendInput call so it appears
	// immediately instead of rendering as paced blocks. Longer transcripts use
	// smaller dispatches so the complete target identity can be revalidated
	// throughout delivery.
	unicodeInputFastPathUnits  = 512
	unicodeInputLongBatchUnits = 256
)

var _ [nativeInputSize - unsafe.Sizeof(inputEvent{})]byte
var _ [unsafe.Sizeof(inputEvent{}) - nativeInputSize]byte

func keyboardEvent(scan uint16, flags uint32) inputEvent {
	event := inputEvent{Type: 1}
	*(*keyInput)(unsafe.Pointer(&event.Data[0])) = keyInput{Scan: scan, Flags: flags}
	return event
}

var insertionForeground = foreground
var insertionModifiersReleased = modifiersReleased

func modifiersReleased() bool {
	// SHIFT, CONTROL and MENU cover both sides; Windows keys have separate VKeys.
	for _, key := range [...]uintptr{0x10, 0x11, 0x12, 0x5B, 0x5C} {
		state, _, _ := getAsyncKeyState.Call(key)
		if state&0x8000 != 0 {
			return false
		}
	}
	return true
}

// SendInput preserves physical modifier state. Never synthesize key-up events:
// wait briefly for release and keep validating the captured target while waiting.
func waitForInsertion(ctx context.Context, target insertion.Target) (string, error) {
	deadline := time.NewTimer(250 * time.Millisecond)
	defer deadline.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return "context", err
		}
		current, err := insertionForeground()
		if err != nil || !current.Valid() || current != target {
			return "focus", insertion.ErrCopyRequired
		}
		if insertionModifiersReleased() {
			return "context", ctx.Err()
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "context", ctx.Err()
		case <-deadline.C:
			timer.Stop()
			return "modifiers", insertion.NewRejection(insertion.Send, "modifiers_held")
		case <-timer.C:
		}
	}
}

var dispatchUnicodeEvents = func(events []inputEvent) uintptr {
	n, _, _ := sendInput.Call(uintptr(len(events)), uintptr(unsafe.Pointer(&events[0])), nativeInputSize)
	runtime.KeepAlive(events)
	return n
}

func (i Input) insertionLogger() *slog.Logger {
	if i.logger == nil {
		return diagnostics.DiscardLogger()
	}
	return i.logger
}

func (i Input) insertionFailed(start time.Time, units, batches int, strategy, stage string, err error) error {
	i.insertionLogger().Warn(
		"direct input failed",
		"utf16_units", units,
		"batch_count", batches,
		"duration_ms", time.Since(start).Milliseconds(),
		"strategy", strategy,
		"stage", stage,
		"error_kind", diagnostics.ErrorKind(err),
	)
	return err
}

func (i Input) InsertUnicode(ctx context.Context, target insertion.Target, text string) error {
	started := time.Now()
	u, e := windows.UTF16FromString(text)
	if e != nil {
		return i.insertionFailed(started, 0, 0, "none", "encoding", e)
	}
	u = u[:len(u)-1]
	if len(u) == 0 {
		return nil
	}
	strategy := "single"
	batchUnits := len(u)
	if len(u) > unicodeInputFastPathUnits {
		strategy = "adaptive"
		batchUnits = unicodeInputLongBatchUnits
	}
	batches := 0
	for offset := 0; offset < len(u); {
		end := min(offset+batchUnits, len(u))
		// Keep a UTF-16 surrogate pair in one dispatch so no Unicode scalar is
		// split across independently accepted SendInput calls.
		if end < len(u) && u[end-1] >= 0xD800 && u[end-1] <= 0xDBFF {
			end--
		}
		events := make([]inputEvent, 0, (end-offset)*2)
		for _, v := range u[offset:end] {
			events = append(events, keyboardEvent(v, 4), keyboardEvent(v, 4|2))
		}
		if stage, err := waitForInsertion(ctx, target); err != nil {
			return i.insertionFailed(started, len(u), batches, strategy, stage, err)
		}
		batches++
		if sent := dispatchUnicodeEvents(events); sent != uintptr(len(events)) {
			partialErr := fmt.Errorf("Unicode insertion was partial or blocked: sent %d of %d events", sent, len(events))
			return i.insertionFailed(started, len(u), batches, strategy, "dispatch", partialErr)
		}
		offset = end
	}
	i.insertionLogger().Info(
		"direct input completed",
		"utf16_units", len(u),
		"batch_count", batches,
		"duration_ms", time.Since(started).Milliseconds(),
		"strategy", strategy,
	)
	return nil
}
