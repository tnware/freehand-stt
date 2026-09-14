//go:build darwin

package managedruntime

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
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
				err = p.wait(ctx)
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

func TestDarwinExitBeforeSupervisorRegistration(t *testing.T) {
	for _, tc := range []struct {
		name         string
		code         int
		invalidQueue bool
	}{{"success", 0, false}, {"nonzero", 7, false}, {"registration_error", 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			kq, err := unix.Kqueue()
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(kq)
			unix.CloseOnExec(kq)
			life, lifeWrite, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer life.Close()
			defer lifeWrite.Close() // Keep lifetime open: exit alone must finish supervision.
			readyRead, ready, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer readyRead.Close()
			defer ready.Close()
			child := exec.Command("/bin/sh", "-c", fmt.Sprintf("/bin/sleep 60 & echo $!; exit %d", tc.code))
			child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			output, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = child.Start(); err != nil {
				t.Fatal(err)
			}
			pid := child.Process.Pid
			reaped := false
			defer func() {
				if !reaped {
					_ = unix.Kill(-pid, unix.SIGKILL)
					_ = child.Wait()
				}
			}()
			_ = output.(*os.File).SetReadDeadline(time.Now().Add(5 * time.Second))
			var leaf int
			if _, err = fmt.Fscanln(output, &leaf); err != nil || leaf <= 0 {
				t.Fatal("descendant PID missing", err)
			}
			// Observe an actual zombie, without Wait (which would release the PID).
			// No scheduling delay is used to manufacture the registration race.
			deadline := time.Now().Add(5 * time.Second)
			for {
				info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
				if err != nil {
					t.Fatal(err)
				}
				if info.Proc.P_stat == 5 { // SZOMB, from <sys/proc.h>.
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("child did not exit before registration")
				}
				runtime.Gosched()
			}
			if err := unix.Kill(leaf, 0); err != nil {
				t.Fatal("descendant did not survive the leader", err)
			}
			queue := kq
			if tc.invalidQueue {
				queue = -1 // EBADF must not be mistaken for an already-exited child.
			}
			watchdog := time.AfterFunc(5*time.Second, func() { _ = lifeWrite.Close() })
			defer watchdog.Stop()
			got := superviseRuntimeChild(queue, child, life, ready)
			reaped = true
			if !watchdog.Stop() {
				t.Fatal("supervisor waited for lifetime EOF after child exit")
			}
			want := tc.code
			if tc.invalidQueue {
				want = 125
			}
			if got != want {
				t.Fatalf("pre-registration exit code = %d, want %d", got, want)
			}
			var message [8]byte
			n, err := io.ReadFull(readyRead, message[:])
			if tc.invalidQueue {
				if n != 0 || err != io.EOF {
					t.Fatalf("registration failure published readiness: %d bytes, %v", n, err)
				}
			} else {
				if err != nil {
					t.Fatal("pre-registration exit did not publish readiness", err)
				}
				if got := binary.LittleEndian.Uint64(message[:]); got != uint64(pid) {
					t.Fatalf("ready PID = %d, want %d", got, pid)
				}
			}
			awaitNativeExit(t, pid)
			awaitNativeExit(t, leaf)
		})
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
