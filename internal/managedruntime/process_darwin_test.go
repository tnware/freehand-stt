//go:build darwin

package managedruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// macOS exposes /var as an OS symlink. Use its physical temporary directory
// for fixtures; production managed storage still rejects linked ancestors.
func TestMain(m *testing.M) {
	dir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil || os.Setenv("TMPDIR", dir) != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestDarwinRuntimeFixture(t *testing.T) {
	switch os.Getenv("FREEHAND_RUNTIME_FIXTURE") {
	case "tree":
		child := exec.Command(os.Args[0], "-test.run=^TestDarwinRuntimeFixture$")
		child.Env = []string{"FREEHAND_RUNTIME_FIXTURE=leaf"}
		if child.Start() != nil {
			os.Exit(2)
		}
		fmt.Printf("%d\n", child.Process.Pid)
		if os.Getenv("FREEHAND_RUNTIME_EXIT") == "1" {
			os.Exit(0)
		}
		time.Sleep(time.Minute)
	case "leaf":
		time.Sleep(time.Minute)
	case "listener":
		l, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			os.Exit(2)
		}
		fmt.Printf("%d\n", l.Addr().(*net.TCPAddr).Port)
		time.Sleep(time.Minute)
	case "parent":
		p, err := launchOwned(context.Background(), os.Args[0], []string{"-test.run=^TestDarwinRuntimeFixture$"}, "", []string{"FREEHAND_RUNTIME_FIXTURE=tree"})
		if err != nil {
			os.Exit(2)
		}
		deadline := time.Now().Add(5 * time.Second)
		for len(p.stdout.bytes()) == 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		leaf, err := strconv.Atoi(stringTrim(p.stdout.bytes()))
		if err != nil {
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode([]int{p.pid, leaf})
		time.Sleep(time.Minute)
	}
}

func stringTrim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return string(b)
}

func awaitNativeNumber(t *testing.T, p *ownedProcess) int {
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

func TestDarwinOwnedTreeCancellationAndNaturalExit(t *testing.T) {
	for _, natural := range []bool{false, true} {
		t.Run(fmt.Sprint(natural), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			env := []string{"FREEHAND_RUNTIME_FIXTURE=tree"}
			if natural {
				env = append(env, "FREEHAND_RUNTIME_EXIT=1")
			}
			p, err := launchOwned(ctx, os.Args[0], []string{"-test.run=^TestDarwinRuntimeFixture$"}, "", env)
			if err != nil {
				t.Fatal(err)
			}
			defer p.kill()
			leaf := awaitNativeNumber(t, p)
			if !natural {
				cancel()
			}
			wait, stop := context.WithTimeout(t.Context(), 5*time.Second)
			defer stop()
			err = p.wait(wait)
			if natural && err != nil {
				t.Fatal("successful child exit was lost", err)
			}
			awaitNativeExit(t, p.pid)
			awaitNativeExit(t, leaf)
		})
	}
}

func TestDarwinImmediateCommandExit(t *testing.T) {
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		p, err := launchOwned(ctx, "/usr/bin/true", nil, "", nil)
		if err == nil {
			err = p.wait(ctx)
		}
		cancel()
		if err != nil {
			t.Fatal("immediate successful exit lost", err)
		}
	}
}

func TestDarwinParentCrashKillsOwnedTree(t *testing.T) {
	parent := exec.Command(os.Args[0], "-test.run=^TestDarwinRuntimeFixture$")
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
	if err = json.NewDecoder(output).Decode(&pids); err != nil || len(pids) != 2 {
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

func TestDarwinListenerOwnership(t *testing.T) {
	p, err := launchOwned(t.Context(), os.Args[0], []string{"-test.run=^TestDarwinRuntimeFixture$"}, "", []string{"FREEHAND_RUNTIME_FIXTURE=listener"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.kill()
	port := awaitNativeNumber(t, p)
	if owned, err := ownsListener(port, p.pid); err != nil || !owned {
		t.Fatalf("owned listener rejected: %v %v", owned, err)
	}
	if owned, err := ownsListener(port, os.Getpid()); err != nil || owned {
		t.Fatalf("foreign listener adopted: %v %v", owned, err)
	}
	p.kill()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_ = p.wait(ctx)
	if owned, _ := ownsListener(port, p.pid); owned {
		t.Fatal("dead listener adopted")
	}
}
