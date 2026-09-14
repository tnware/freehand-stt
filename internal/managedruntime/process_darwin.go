//go:build darwin

package managedruntime

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const supervisorArgument = "--freehand-runtime-supervisor"

// Re-exec before application startup. The helper receives only the already
// admitted executable/arguments/environment and two private inherited pipes.
// It never starts Wails, accesses settings, or accepts renderer/terminal input.
func init() {
	if len(os.Args) > 1 && os.Args[1] == supervisorArgument {
		os.Exit(runRuntimeSupervisor())
	}
}

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
	kq, err := unix.Kqueue()
	if err != nil {
		return 125
	}
	defer unix.Close(kq)
	unix.CloseOnExec(kq)
	child := exec.Command(os.Args[2], os.Args[3:]...)
	child.Env = os.Environ()
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if child.Start() != nil {
		return 125
	}
	pid := child.Process.Pid
	// Observe exit without reaping the group leader. Its reserved PID prevents
	// group-ID reuse until descendants have been killed and Wait reaps it.
	changes := []unix.Kevent_t{
		{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ENABLE, Fflags: unix.NOTE_EXIT},
		{Ident: uint64(life.Fd()), Filter: unix.EVFILT_READ, Flags: unix.EV_ADD | unix.EV_ENABLE},
	}
	_, err = unix.Kevent(kq, changes, nil, nil)
	if err == nil {
		var message [8]byte
		binary.LittleEndian.PutUint64(message[:], uint64(pid))
		_, err = ready.Write(message[:])
	}
	ready.Close()
	if err == nil {
		events := make([]unix.Kevent_t, 2)
		for {
			_, err = unix.Kevent(kq, nil, events, nil)
			if err != unix.EINTR {
				break
			}
		}
	}
	// Lifetime-pipe EOF covers Stop, cancellation, Quit, and parent SIGKILL.
	// Natural child exit also closes any surviving descendants (e.g. curl).
	_ = unix.Kill(-pid, unix.SIGKILL)
	waitErr := child.Wait()
	if err != nil {
		return 125
	}
	if waitErr != nil {
		if code := child.ProcessState.ExitCode(); code >= 0 {
			return code
		}
		return 125
	}
	return 0
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
