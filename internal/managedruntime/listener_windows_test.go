//go:build windows

package managedruntime

import (
	"net"
	"os"
	"testing"
)

func TestWindowsListenerOwnership(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	owned, err := ownsListener(port, os.Getpid())
	if err != nil || !owned {
		t.Fatalf("own loopback listener not recognized: %v %v", owned, err)
	}
	owned, err = ownsListener(port, 0)
	if err == nil && owned {
		t.Fatal("accepted another process as owner")
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	owned, err = ownsListener(port, os.Getpid())
	if err != nil || owned {
		t.Fatalf("closed listener accepted: %v %v", owned, err)
	}
}

func TestListenerTableRejectsMalformedAndNonLoopback(t *testing.T) {
	for _, table := range [][]byte{nil, {1, 0, 0}, {1, 0, 0, 0}} {
		if _, err := listenerInTable(table, 1234, 99); err == nil {
			t.Fatal("accepted truncated table")
		}
	}
	// MIB_TCPTABLE_OWNER_PID: count, followed by six DWORDs per row.
	table := []byte{1, 0, 0, 0, 2, 0, 0, 0, 127, 0, 0, 1, 4, 210, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 99, 0, 0, 0}
	if owned, err := listenerInTable(table, 1234, 99); err != nil || !owned {
		t.Fatalf("valid table: %v %v", owned, err)
	}
	table[8] = 0
	table[11] = 0
	if owned, err := listenerInTable(table, 1234, 99); err != nil || owned {
		t.Fatalf("accepted wildcard interface: %v %v", owned, err)
	}
}
