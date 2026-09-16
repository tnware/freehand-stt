//go:build windows

package input

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tnware/freehand-stt/internal/insertion"
	"golang.org/x/sys/windows"
)

// Fixture operations never read or modify the desktop clipboard. They still
// check real thread identity across scheduling points in the production path.
type clipboardFixture struct {
	t       *testing.T
	thread  uint32
	calls   []string
	text    []uint16
	fail    string
	onOpen  func()
	onEmpty func()
}

func (f *clipboardFixture) call(name string) {
	f.t.Helper()
	runtime.Gosched()
	thread := windows.GetCurrentThreadId()
	if f.thread == 0 {
		f.thread = thread
	}
	if f.thread != thread {
		f.t.Fatalf("clipboard thread changed from %d to %d", f.thread, thread)
	}
	f.calls = append(f.calls, name)
}
func (f *clipboardFixture) createOwner() (uintptr, error) {
	f.call("owner")
	if f.fail == "owner" {
		return 0, errors.New("fixture owner failure")
	}
	return 7, nil
}
func (f *clipboardFixture) destroyOwner(owner uintptr) {
	f.call("destroy")
	if owner != 7 {
		f.t.Fatal("wrong owner")
	}
}
func (f *clipboardFixture) allocateText(text []uint16) (uintptr, error) {
	f.call("allocate")
	f.text = slices.Clone(text)
	if f.fail == "allocate" {
		return 0, errors.New("fixture allocation failure")
	}
	return 11, nil
}
func (f *clipboardFixture) freeText(handle uintptr) {
	f.call("free")
	if handle != 11 {
		f.t.Fatal("wrong handle")
	}
}
func (f *clipboardFixture) open(owner uintptr) bool {
	f.call("open")
	if owner != 7 {
		f.t.Fatal("clipboard requires the owned HWND")
	}
	if f.onOpen != nil {
		f.onOpen()
	}
	return f.fail != "open"
}
func (f *clipboardFixture) close() { f.call("close") }
func (f *clipboardFixture) empty() error {
	f.call("empty")
	if f.onEmpty != nil {
		f.onEmpty()
	}
	if f.fail == "empty" {
		return errors.New("fixture empty failure")
	}
	return nil
}
func (f *clipboardFixture) setText(handle uintptr) error {
	f.call("set")
	if handle != 11 {
		f.t.Fatal("wrong handle")
	}
	if f.fail == "set" {
		return errors.New("fixture set failure")
	}
	return nil
}

func TestClipboardTransactionOwnsWindowThreadAndTransfersUnicode(t *testing.T) {
	f := &clipboardFixture{t: t}
	if err := copyClipboard(t.Context(), "A😀文", f); err != nil {
		t.Fatal(err)
	}
	if want := []string{"owner", "allocate", "open", "empty", "set", "close", "destroy"}; !slices.Equal(f.calls, want) {
		t.Fatalf("calls=%v, want %v", f.calls, want)
	}
	if want := []uint16{65, 0xD83D, 0xDE00, 0x6587, 0}; !slices.Equal(f.text, want) {
		t.Fatalf("UTF-16 payload=%v", f.text)
	}
}

func TestClipboardFailuresKeepMemoryOwnershipAndDistinguishMutation(t *testing.T) {
	for _, failure := range []string{"owner", "allocate", "open", "empty", "set"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := &clipboardFixture{t: t, fail: failure}
				err := copyClipboard(t.Context(), "text", f)
				want := insertion.ErrCopyNotPerformed
				if failure == "set" {
					want = insertion.ErrCopyFailed
				}
				if !errors.Is(err, want) {
					t.Fatalf("error=%v, want %v", err, want)
				}
				allocated := failure != "owner" && failure != "allocate"
				if slices.Contains(f.calls, "free") != allocated {
					t.Fatalf("allocation ownership: %v", f.calls)
				}
				opened := failure == "empty" || failure == "set"
				if slices.Contains(f.calls, "close") != opened {
					t.Fatalf("open ownership: %v", f.calls)
				}
				if failure == "open" && slices.Contains(f.calls, "empty") {
					t.Fatal("changed busy clipboard")
				}
			})
		})
	}
}

func TestClipboardCancellationBeforeAndAfterDestructiveBoundary(t *testing.T) {
	for _, afterEmpty := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		f := &clipboardFixture{t: t}
		if afterEmpty {
			f.onEmpty = cancel
		} else {
			f.onOpen = cancel
		}
		err := copyClipboard(ctx, "text", f)
		cancel()
		if afterEmpty {
			if err != nil || !slices.Contains(f.calls, "set") || slices.Contains(f.calls, "free") {
				t.Fatalf("must finish ownership transfer: error=%v calls=%v", err, f.calls)
			}
		} else if !errors.Is(err, context.Canceled) || !errors.Is(err, insertion.ErrCopyNotPerformed) || slices.Contains(f.calls, "empty") {
			t.Fatalf("canceled copy changed clipboard: error=%v calls=%v", err, f.calls)
		}
	}
}

func TestClipboardGateWaitIsBoundedAndDoesNotCreateOwner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		oldGate := clipboardGate
		clipboardGate = make(chan struct{}, 1)
		defer func() { clipboardGate = oldGate }()
		clipboardGate <- struct{}{}
		defer func() { <-clipboardGate }()
		f := &clipboardFixture{t: t}
		start := time.Now()
		err := copyClipboard(t.Context(), "text", f)
		if !errors.Is(err, context.DeadlineExceeded) || len(f.calls) != 0 || time.Since(start) != 500*time.Millisecond {
			t.Fatalf("error=%v calls=%v elapsed=%s", err, f.calls, time.Since(start))
		}
	})
}

func TestNativeClipboardOwnerUsesCurrentThreadWithoutOpeningClipboard(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	b := windowsClipboard{}
	owner, err := b.createOwner()
	if err != nil {
		t.Fatal(err)
	}
	defer b.destroyOwner(owner)
	thread, _, _ := getWindowThreadProcessID.Call(owner, 0)
	if uint32(thread) != windows.GetCurrentThreadId() {
		t.Fatal("owner belongs to another thread")
	}
	// Exercise allocation/cleanup too, without ever opening or changing clipboard.
	text, _ := windows.UTF16FromString("allocation 😀")
	handle, err := b.allocateText(text)
	if err != nil {
		t.Fatal(err)
	}
	b.freeText(handle)
}
