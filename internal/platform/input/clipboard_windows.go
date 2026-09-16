//go:build windows

package input

import (
	"context"
	"errors"
	"runtime"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/insertion"
	"golang.org/x/sys/windows"
)

var (
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	createWindowEx   = user32.NewProc("CreateWindowExW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
	globalFree       = kernel32.NewProc("GlobalFree")
	moveMemory       = kernel32.NewProc("RtlMoveMemory")
	clipboardGate    = make(chan struct{}, 1)
)

type clipboardBackend interface {
	createOwner() (uintptr, error)
	destroyOwner(uintptr)
	allocateText([]uint16) (uintptr, error)
	freeText(uintptr)
	open(uintptr) bool
	close()
	empty() error
	setText(uintptr) error
}

type windowsClipboard struct{}

func (windowsClipboard) createOwner() (uintptr, error) {
	class := windows.StringToUTF16Ptr("STATIC")
	// HWND_MESSAGE (-3) creates an invisible, nonactivating owner. A predefined
	// class needs no Go window callback, persistent message loop or app HWND.
	h, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if h == 0 {
		return 0, errors.New("clipboard owner could not be created")
	}
	return h, nil
}

func (windowsClipboard) destroyOwner(owner uintptr) { destroyWindow.Call(owner) }

func (windowsClipboard) allocateText(text []uint16) (uintptr, error) {
	size := uintptr(len(text) * 2)
	h, _, _ := globalAlloc.Call(0x0002, size) // GMEM_MOVEABLE; ownership transfers on SetClipboardData.
	if h == 0 {
		return 0, errors.New("clipboard memory could not be allocated")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		globalFree.Call(h)
		return 0, errors.New("clipboard memory could not be locked")
	}
	// p is native allocation memory, never a Go address stored as uintptr.
	// Copy through the native ABI instead of fabricating a Go slice over it.
	moveMemory.Call(p, uintptr(unsafe.Pointer(&text[0])), size)
	runtime.KeepAlive(text)
	globalUnlock.Call(h)
	return h, nil
}

func (windowsClipboard) freeText(handle uintptr) { globalFree.Call(handle) }
func (windowsClipboard) open(owner uintptr) bool {
	r, _, _ := openClipboard.Call(owner)
	return r != 0
}
func (windowsClipboard) close() { closeClipboard.Call() }
func (windowsClipboard) empty() error {
	if r, _, _ := emptyClipboard.Call(); r == 0 {
		return errors.New("clipboard could not be emptied")
	}
	return nil
}
func (windowsClipboard) setText(handle uintptr) error {
	if r, _, _ := setClipboardData.Call(13, handle); r == 0 { // CF_UNICODETEXT
		return errors.New("clipboard text could not be set")
	}
	return nil
}

func (Input) Copy(ctx context.Context, text string) error {
	return copyClipboard(ctx, text, windowsClipboard{})
}

func copyClipboard(ctx context.Context, text string, backend clipboardBackend) error {
	u, err := windows.UTF16FromString(text)
	if err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	// Bound contention without spawning waiters. Copies share a gate even when
	// Input values have been copied into separate feature dependencies.
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	select {
	case clipboardGate <- struct{}{}:
		defer func() { <-clipboardGate }()
	case <-ctx.Done():
		return errors.Join(insertion.ErrCopyNotPerformed, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	// Open/Empty/Set/Close and the owner window must belong to the same OS
	// thread. Everything is synchronous; no clipboard worker survives Copy.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	owner, err := backend.createOwner()
	if err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	defer backend.destroyOwner(owner)
	handle, err := backend.allocateText(u)
	if err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	transferred := false
	defer func() {
		if !transferred {
			backend.freeText(handle)
		}
	}()
	opened := false
	for attempt := range 8 {
		if err := ctx.Err(); err != nil {
			return errors.Join(insertion.ErrCopyNotPerformed, err)
		}
		if backend.open(owner) {
			opened = true
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(insertion.ErrCopyNotPerformed, ctx.Err())
		case <-timer.C:
		}
	}
	if !opened {
		return errors.Join(insertion.ErrCopyNotPerformed, errors.New("could not open clipboard"))
	}
	defer backend.close()
	// Cancellation is honored until the destructive boundary. After Empty,
	// finish Set synchronously and report whether clipboard state changed.
	if err := ctx.Err(); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	if err := backend.empty(); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	if err := backend.setText(handle); err != nil {
		return errors.Join(insertion.ErrCopyFailed, err)
	}
	transferred = true
	return nil
}
