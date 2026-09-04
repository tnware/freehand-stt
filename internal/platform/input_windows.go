//go:build windows

package platform

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
var openClipboard = user32.NewProc("OpenClipboard")
var closeClipboard = user32.NewProc("CloseClipboard")
var emptyClipboard = user32.NewProc("EmptyClipboard")
var setClipboardData = user32.NewProc("SetClipboardData")
var globalAlloc = kernel32.NewProc("GlobalAlloc")
var globalLock = kernel32.NewProc("GlobalLock")
var globalUnlock = kernel32.NewProc("GlobalUnlock")
var globalFree = kernel32.NewProc("GlobalFree")

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
		select {
		case <-ctx.Done():
			return i.insertionFailed(started, len(u), batches, strategy, "context", ctx.Err())
		default:
		}
		current, foregroundErr := insertionForeground()
		if foregroundErr != nil || !current.Valid() || current != target {
			return i.insertionFailed(started, len(u), batches, strategy, "focus", insertion.ErrCopyRequired)
		}
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
func (Input) Copy(ctx context.Context, text string) error {
	u, e := windows.UTF16FromString(text)
	if e != nil {
		return e
	}
	size := uintptr(len(u) * 2)
	h, _, e := globalAlloc.Call(0x0002, size)
	if h == 0 {
		return e
	}
	transferred := false
	defer func() {
		if !transferred {
			globalFree.Call(h)
		}
	}()
	p, _, e := globalLock.Call(h)
	if p == 0 {
		return e
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u))
	copy(dst, u)
	globalUnlock.Call(h)
	var opened bool
	for i := 0; i < 8; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		r, _, _ := openClipboard.Call(0)
		if r != 0 {
			opened = true
			break
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	if !opened {
		return errors.New("could not open clipboard")
	}
	defer closeClipboard.Call()
	if r, _, e := emptyClipboard.Call(); r == 0 {
		return e
	}
	if r, _, e := setClipboardData.Call(13, h); r == 0 {
		return e
	}
	transferred = true
	return nil
}
