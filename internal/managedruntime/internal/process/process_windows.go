//go:build windows

package process

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Create suspended: no child code (including curl creation) executes until the
// process has been assigned to our non-inheritable kill-on-close Job Object.
func startOwnedProcess(cmd *exec.Cmd) (func(), int, error) {
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return nil, 0, e
	}
	closeJob := func() { windows.CloseHandle(job) }
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); e != nil {
		closeJob()
		return nil, 0, e
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_SUSPENDED | windows.CREATE_NO_WINDOW}
	if e = cmd.Start(); e != nil {
		closeJob()
		return nil, 0, e
	}
	fail := func(err error) (func(), int, error) { cmd.Process.Kill(); cmd.Wait(); closeJob(); return nil, 0, err }
	proc, e := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if e != nil {
		return fail(e)
	}
	e = windows.AssignProcessToJobObject(job, proc)
	windows.CloseHandle(proc)
	if e != nil {
		return fail(e)
	}
	snapshot, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if e != nil {
		return fail(e)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for e = windows.Thread32First(snapshot, &entry); e == nil; e = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != uint32(cmd.Process.Pid) {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return fail(err)
		}
		_, err = windows.ResumeThread(thread)
		windows.CloseHandle(thread)
		if err != nil {
			return fail(err)
		}
		return closeJob, cmd.Process.Pid, nil
	}
	return fail(windows.ERROR_NOT_FOUND)
}
