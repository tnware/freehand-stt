//go:build linux

package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLinuxRuntimeFixture(t *testing.T) {
	switch os.Getenv("FREEHAND_RUNTIME_FIXTURE") {
	case "tree":
		child := exec.Command(os.Args[0], "-test.run=^TestLinuxRuntimeFixture$")
		child.Env = []string{"FREEHAND_RUNTIME_FIXTURE=leaf"}
		// Escape the original group/session: subreaper ownership must still hold.
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		out, err := child.StdoutPipe()
		if err != nil {
			os.Exit(2)
		}
		if child.Start() != nil {
			os.Exit(2)
		}
		var grandchild int
		if _, err := fmt.Fscanln(out, &grandchild); err != nil {
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode([]int{child.Process.Pid, grandchild})
		if os.Getenv("FREEHAND_RUNTIME_EXIT") == "1" {
			os.Exit(0)
		}
		time.Sleep(time.Minute)
	case "leaf":
		grandchild := exec.Command(os.Args[0], "-test.run=^TestLinuxRuntimeFixture$")
		grandchild.Env = []string{"FREEHAND_RUNTIME_FIXTURE=grandchild"}
		grandchild.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if grandchild.Start() != nil {
			os.Exit(2)
		}
		fmt.Printf("%d\n", grandchild.Process.Pid)
		time.Sleep(time.Minute)
	case "grandchild":
		time.Sleep(time.Minute)
	case "listener":
		l, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			os.Exit(2)
		}
		fmt.Printf("%d\n", l.Addr().(*net.TCPAddr).Port)
		time.Sleep(time.Minute)
	case "parent":
		p, err := launchOwned(t.Context(), os.Args[0], []string{"-test.run=^TestLinuxRuntimeFixture$"}, "", []string{"FREEHAND_RUNTIME_FIXTURE=tree"})
		if err != nil {
			os.Exit(2)
		}
		deadline := time.Now().Add(5 * time.Second)
		for len(p.stdout.bytes()) == 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		pids := awaitNativePIDs(t, p)
		_ = json.NewEncoder(os.Stdout).Encode(append([]int{p.pid}, pids...))
		time.Sleep(time.Minute)
	}
}

func stringTrim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return string(b)
}

func awaitNativeNumber(t *testing.T, p *Process) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if n, err := strconv.Atoi(stringTrim(p.stdout.bytes())); err == nil {
			return n
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("native fixture did not become ready")
	return 0
}

func awaitNativeExit(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if unix.Kill(pid, 0) == unix.ESRCH {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("owned process %d survived", pid)
}

func TestLinuxOwnedTreeCancellationAndNaturalExit(t *testing.T) {
	for _, mode := range []string{"cancel", "stop", "natural"} {
		natural := mode == "natural"
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			env := []string{"FREEHAND_RUNTIME_FIXTURE=tree"}
			if natural {
				env = append(env, "FREEHAND_RUNTIME_EXIT=1")
			}
			p, err := launchOwned(ctx, os.Args[0], []string{"-test.run=^TestLinuxRuntimeFixture$"}, "", env)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Kill()
			pids := awaitNativePIDs(t, p)
			if mode == "cancel" {
				cancel()
			}
			if mode == "stop" {
				p.Kill()
				p.Kill()
			}
			wait, stop := context.WithTimeout(t.Context(), 5*time.Second)
			defer stop()
			err = p.Wait(wait)
			if natural && err != nil {
				t.Fatal("successful child exit was lost", err)
			}
			awaitNativeExit(t, p.pid)
			for _, pid := range pids {
				awaitNativeExit(t, pid)
			}
		})
	}
}

func TestLinuxImmediateCommandExit(t *testing.T) {
	for _, tc := range []struct {
		exe  string
		code int
	}{{"/usr/bin/true", 0}, {"/usr/bin/false", 1}} {
		t.Run(filepath.Base(tc.exe), func(t *testing.T) {
			for i := 0; i < 10; i++ {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				p, err := launchOwned(ctx, tc.exe, nil, "", nil)
				if err != nil {
					cancel()
					t.Fatal("immediate exit failed supervision", err)
				}
				err = p.Wait(ctx)
				cancel()
				if tc.code == 0 {
					if err != nil {
						t.Fatal("immediate successful exit lost", err)
					}
				} else {
					var exitErr *exec.ExitError
					if !errors.As(err, &exitErr) || exitErr.ExitCode() != tc.code {
						t.Fatalf("immediate exit code %d lost: %v", tc.code, err)
					}
				}
			}
		})
	}
}

func TestLinuxParentCrashKillsOwnedTree(t *testing.T) {
	parent := exec.Command(os.Args[0], "-test.run=^TestLinuxRuntimeFixture$")
	parent.Env = []string{"FREEHAND_RUNTIME_FIXTURE=parent"}
	output, err := parent.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = parent.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = parent.Process.Kill(); _ = parent.Wait() })
	_ = output.(*os.File).SetReadDeadline(time.Now().Add(10 * time.Second))
	var pids []int
	if err = json.NewDecoder(output).Decode(&pids); err != nil || len(pids) != 3 {
		t.Fatal("parent did not establish its owned tree", err)
	}
	if err = parent.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = parent.Wait()
	for _, pid := range pids {
		awaitNativeExit(t, pid)
	}
}

func TestLinuxListenerOwnership(t *testing.T) {
	p, err := launchOwned(t.Context(), os.Args[0], []string{"-test.run=^TestLinuxRuntimeFixture$"}, "", []string{"FREEHAND_RUNTIME_FIXTURE=listener"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Kill()
	port := awaitNativeNumber(t, p)
	if owned, err := OwnsListener(port, p.pid); err != nil || !owned {
		t.Fatalf("owned listener rejected: %v %v", owned, err)
	}
	if owned, err := OwnsListener(port, os.Getpid()); err != nil || owned {
		t.Fatalf("foreign listener adopted: %v %v", owned, err)
	}
	p.Kill()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_ = p.Wait(ctx)
	if owned, _ := OwnsListener(port, p.pid); owned {
		t.Fatal("dead listener adopted")
	}
}

func awaitNativePIDs(t *testing.T, p *Process) []int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var pids []int
		if json.Unmarshal(p.stdout.bytes(), &pids) == nil && len(pids) == 2 {
			return pids
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("native tree fixture did not become ready")
	return nil
}

func TestLinuxListenerRejectsWildcardAndInvalidIdentity(t *testing.T) {
	l, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if owned, err := OwnsListener(l.Addr().(*net.TCPAddr).Port, os.Getpid()); err != nil || owned {
		t.Fatalf("wildcard listener admitted: %v %v", owned, err)
	}
	for _, port := range []int{0, -1, 65536} {
		if owned, err := OwnsListener(port, os.Getpid()); owned || err == nil {
			t.Fatal("invalid port admitted")
		}
	}
	if owned, err := OwnsListener(1234, 0); owned || err == nil {
		t.Fatal("invalid PID admitted")
	}
}
