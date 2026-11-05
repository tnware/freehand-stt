//go:build linux

package process

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const supervisorArgument = "--freehand-runtime-supervisor"

func init() {
	if len(os.Args) > 1 && os.Args[1] == supervisorArgument {
		os.Exit(runRuntimeSupervisor())
	}
}

// The private re-exec owns all descendants, including children that create a
// new session. Lifetime EOF also works when the application receives SIGKILL.
// A subreaper is essential: a process group alone cannot contain daemonization.
func runRuntimeSupervisor() int {
	if len(os.Args) < 3 || !filepath.IsAbs(os.Args[2]) {
		return 125
	}
	life, ready := os.NewFile(3, "lifetime"), os.NewFile(4, "ready")
	for _, f := range []*os.File{life, ready} {
		st, err := f.Stat()
		if err != nil || st.Mode()&os.ModeNamedPipe == 0 {
			return 125
		}
		unix.CloseOnExec(int(f.Fd()))
	}
	if unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) != nil {
		return 125
	}
	// Refuse launch if the kernel ownership mechanisms are unavailable.
	probe, err := unix.PidfdOpen(os.Getpid(), 0)
	if err != nil {
		return 125
	}
	unix.Close(probe)
	if _, err := supervisorChildren(); err != nil {
		return 125
	}
	child := exec.Command(os.Args[2], os.Args[3:]...)
	child.Env = os.Environ()
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if child.Start() != nil {
		return 125
	}
	pid := child.Process.Pid
	fd, err := unix.PidfdOpen(pid, 0)
	if err == nil {
		var message [8]byte
		binary.LittleEndian.PutUint64(message[:], uint64(pid))
		_, err = ready.Write(message[:])
	}
	ready.Close()
	if err == nil {
		events := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}, {Fd: int32(life.Fd()), Events: unix.POLLIN}}
		for {
			_, err = unix.Poll(events, -1)
			if err != unix.EINTR {
				break
			}
		}
	}
	if fd >= 0 {
		unix.Close(fd)
	}
	code := reapRuntimeTree(pid)
	_ = child.Process.Release()
	if err != nil {
		return 125
	}
	return code
}

// No other goroutine calls Wait. Every PID read here is our direct child and
// remains reserved, even after exit, until this routine reaps it. Thus signals
// cannot hit a reused PID. Killing parents adopts their descendants, including
// setsid/double-fork workers, and ECHILD is the only successful stopping point.
func reapRuntimeTree(root int) int {
	code := 125
	for {
		children, err := supervisorChildren()
		if err != nil {
			// The supervisor must not abandon ownership on a transient procfs error.
			time.Sleep(10 * time.Millisecond)
			continue
		}
		for _, pid := range children {
			_ = unix.Kill(pid, unix.SIGKILL)
		}
		for {
			var status unix.WaitStatus
			pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
			if err == unix.EINTR {
				continue
			}
			if err == unix.ECHILD {
				return code
			}
			if err != nil || pid == 0 {
				break
			}
			if pid == root && status.Exited() {
				code = status.ExitStatus()
			}
		}
		time.Sleep(time.Millisecond)
	}
}

func supervisorChildren() ([]int, error) {
	// /proc/PID/task/TID/children depends on CONFIG_CHECKPOINT_RESTORE and
	// is absent on otherwise supported Linux kernels. PPid is always available.
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	parent := strconv.Itoa(os.Getpid())
	var children []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		b, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if os.IsNotExist(err) {
			continue
		}
		// Other users' processes may be invisible under hidepid; ours may not.
		if os.IsPermission(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		end := strings.LastIndexByte(string(b), ')')
		if end < 0 {
			return nil, errors.New("invalid process identity")
		}
		fields := strings.Fields(string(b[end+1:]))
		if len(fields) < 2 {
			return nil, errors.New("invalid process identity")
		}
		if fields[1] == parent {
			children = append(children, pid)
		}
	}
	return children, nil
}

func startOwnedProcess(cmd *exec.Cmd) (func(), int, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, 0, err
	}
	lifeRead, lifeWrite, err := os.Pipe()
	if err != nil {
		return nil, 0, err
	}
	defer lifeRead.Close()
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		lifeWrite.Close()
		return nil, 0, err
	}
	defer readyRead.Close()
	defer readyWrite.Close()
	cmd.Args = append([]string{self, supervisorArgument, cmd.Path}, cmd.Args[1:]...)
	cmd.Path = self
	cmd.ExtraFiles = []*os.File{lifeRead, readyWrite}
	if err = cmd.Start(); err != nil {
		lifeWrite.Close()
		return nil, 0, err
	}
	lifeRead.Close()
	readyWrite.Close()
	_ = readyRead.SetReadDeadline(time.Now().Add(5 * time.Second))
	var message [8]byte
	_, err = io.ReadFull(readyRead, message[:])
	pid := binary.LittleEndian.Uint64(message[:])
	if err != nil || pid == 0 || pid > 1<<31-1 {
		lifeWrite.Close()
		_ = cmd.Wait()
		return nil, 0, errors.New("Could not supervise the managed runtime process.")
	}
	return func() { _ = lifeWrite.Close() }, int(pid), nil
}
