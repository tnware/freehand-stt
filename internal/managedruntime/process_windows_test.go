//go:build windows

package managedruntime

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOwnedHelperProcess(t *testing.T) {
	if os.Getenv("FREEHAND_RUNTIME_HELPER") == "" {
		return
	}
	if os.Getenv("FREEHAND_RUNTIME_DESCENDANT") == "1" {
		for {
			time.Sleep(time.Second)
		}
	}
	exe, _ := os.Executable()
	c := exec.Command(exe, "-test.run=TestOwnedHelperProcess")
	c.Env = append(os.Environ(), "FREEHAND_RUNTIME_DESCENDANT=1")
	if err := c.Start(); err != nil {
		os.Exit(3)
	}
	fmt.Printf("%d\n", c.Process.Pid)
	fmt.Fprint(os.Stderr, strings.Repeat("private output", 100000))
	for {
		time.Sleep(time.Second)
	}
}
func TestWindowsOwnedTreeCancellation(t *testing.T) {
	exe, _ := os.Executable()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := launchOwned(ctx, exe, []string{"-test.run=TestOwnedHelperProcess"}, t.TempDir(), append(os.Environ(), "FREEHAND_RUNTIME_HELPER=1"))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	pid := 0
	for time.Now().Before(deadline) {
		b := c.stdout.bytes()
		if i := strings.IndexByte(string(b), '\n'); i >= 0 {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b[:i])))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		c.kill()
		t.Fatal("helper did not spawn")
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	cancel()
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
		t.Fatal("root survived cancellation")
	}
	if r, err := windows.WaitForSingleObject(h, 3000); err != nil || r != windows.WAIT_OBJECT_0 {
		t.Fatalf("descendant survived job: %d %v", r, err)
	}
	if len(c.stderr.bytes()) > outputLimit {
		t.Fatal("unbounded diagnostics")
	}
	c.kill()
	c.kill()
}
