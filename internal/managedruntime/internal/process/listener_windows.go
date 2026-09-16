//go:build windows

package process

import (
	"encoding/binary"
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

// OwnsListener verifies the child, not just a responding port. An ephemeral
// port chosen by bind/close can be claimed by another process before NeMo binds.
// IPv4 owner-PID table layout follows the Windows IP Helper API contract.
func OwnsListener(port, pid int) (bool, error) {
	if port < 1 || port > 65535 || pid < 1 {
		return false, errors.New("invalid managed listener identity")
	}
	const ownerPIDListener = 3
	size := uint32(4096)
	for attempt := 0; attempt < 4; attempt++ {
		if size < 4 || size > 4<<20 {
			return false, errors.New("managed listener table exceeds its limit")
		}
		buf := make([]byte, size)
		result, _, _ := getExtendedTCPTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, windows.AF_INET, ownerPIDListener, 0)
		if result == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if result != 0 {
			return false, errors.New("could not verify managed listener ownership")
		}
		if int(size) > len(buf) {
			return false, errors.New("invalid managed listener table size")
		}
		return listenerInTable(buf[:size], port, pid)
	}
	return false, errors.New("managed listener table changed during verification")
}

func listenerInTable(table []byte, port, pid int) (bool, error) {
	const rowSize = 24 // six DWORD fields in MIB_TCPROW_OWNER_PID
	if len(table) < 4 {
		return false, errors.New("invalid managed listener table")
	}
	count := binary.LittleEndian.Uint32(table[:4])
	if uint64(count) > uint64((len(table)-4)/rowSize) {
		return false, errors.New("truncated managed listener table")
	}
	for i := uint32(0); i < count; i++ {
		row := table[4+int(i)*rowSize : 4+int(i+1)*rowSize]
		if binary.LittleEndian.Uint32(row[:4]) != 2 {
			continue
		} // LISTEN
		if row[4] != 127 || row[5] != 0 || row[6] != 0 || row[7] != 1 {
			continue
		}
		if int(binary.BigEndian.Uint16(row[8:10])) == port && int(binary.LittleEndian.Uint32(row[20:24])) == pid {
			return true, nil
		}
	}
	return false, nil
}
