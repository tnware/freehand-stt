//go:build linux

package process

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// OwnsListener admits only a TCP IPv4 loopback listener whose socket inode is
// held by the actual server PID, not another process in its network namespace.
// The open proc directory pins this process identity across PID reuse.
func OwnsListener(port, pid int) (bool, error) {
	if port < 1 || port > 65535 || pid < 1 {
		return false, errors.New("invalid managed listener identity")
	}
	dir, err := os.Open("/proc/" + strconv.Itoa(pid))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.New("could not verify managed listener ownership")
	}
	defer dir.Close()
	fds, err := openProcFile(dir, "fd", unix.O_DIRECTORY)
	if err != nil {
		return false, err
	}
	defer fds.Close()
	names, err := fds.Readdirnames(65537)
	if err != nil && err != io.EOF {
		return false, err
	}
	if len(names) > 65536 {
		return false, errors.New("managed descriptor snapshot exceeds limit")
	}
	sockets := make(map[string]bool)
	var link [128]byte
	for _, name := range names {
		n, err := unix.Readlinkat(int(fds.Fd()), name, link[:])
		if err == unix.ENOENT {
			continue
		}
		if err != nil {
			return false, err
		}
		target := string(link[:n])
		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			sockets[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")] = true
		}
	}
	tcp, err := openProcFile(dir, "net/tcp", 0)
	if err != nil {
		return false, err
	}
	defer tcp.Close()
	const maxTable = 4 << 20
	data, err := io.ReadAll(io.LimitReader(tcp, maxTable+1))
	if err != nil || len(data) > maxTable {
		return false, errors.New("could not read bounded managed listener table")
	}
	want := fmt.Sprintf("0100007F:%04X", port)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	owned := false
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 10 && fields[1] == want && fields[3] == "0A" && sockets[fields[9]] {
			owned = true
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	// An exited process must not be admitted from its former namespace table.
	stat, err := openProcFile(dir, "stat", 0)
	if err != nil {
		return false, err
	}
	defer stat.Close()
	b, err := io.ReadAll(io.LimitReader(stat, 8192))
	if err != nil {
		return false, err
	}
	end := strings.LastIndexByte(string(b), ')')
	if end < 0 || end+2 >= len(b) || b[end+2] == 'Z' || b[end+2] == 'X' {
		return false, nil
	}
	return owned, nil
}

func openProcFile(dir *os.File, name string, flags int) (*os.File, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|flags, 0)
	if err != nil {
		return nil, errors.New("could not verify managed listener ownership")
	}
	return os.NewFile(uintptr(fd), name), nil
}
