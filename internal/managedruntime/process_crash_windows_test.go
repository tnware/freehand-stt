//go:build windows

package managedruntime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// This helper is deliberately outside the job it owns. Killing the owner must
// release its non-inheritable job handle and kill both levels below it.
func TestJobOwnerCrashHelper(t *testing.T) {
	if os.Getenv("FREEHAND_JOB_OWNER_HELPER") != "1" {
		return
	}
	exe, _ := os.Executable()
	child, err := launchOwned(context.Background(), exe, []string{"-test.run=^TestOwnedHelperProcess$"}, os.Getenv("TEMP"), append(os.Environ(), "FREEHAND_RUNTIME_HELPER=1"))
	if err != nil {
		os.Exit(4)
	}
	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			child.kill()
			os.Exit(5)
		case <-ticker.C:
			text := string(child.stdout.bytes())
			if strings.Contains(text, "\n") {
				fmt.Printf("%d %s", child.pid, text)
				for {
					time.Sleep(time.Second)
				}
			}
		}
	}
}

func TestWindowsJobKillsTreeOnOwnerCrash(t *testing.T) {
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	owner := exec.CommandContext(ctx, exe, "-test.run=^TestJobOwnerCrashHelper$")
	owner.Env = append(os.Environ(), "FREEHAND_JOB_OWNER_HELPER=1")
	out, err := owner.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Process.Kill(); _ = owner.Wait() }()
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal("owner did not publish process tree")
	}
	parts := strings.Fields(line)
	if len(parts) != 2 {
		t.Fatal("unexpected owner handshake")
	}
	handles := []windows.Handle{}
	for _, part := range parts {
		pid, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)
		handles = append(handles, handle)
	}
	if err := owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = owner.Wait()
	for _, handle := range handles {
		if result, err := windows.WaitForSingleObject(handle, 4000); err != nil || result != windows.WAIT_OBJECT_0 {
			t.Fatalf("owned process survived owner crash: %d %v", result, err)
		}
	}
}
