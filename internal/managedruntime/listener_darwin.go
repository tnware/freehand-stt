//go:build darwin && cgo

package managedruntime

/*
#include <libproc.h>
#include <sys/proc_info.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <errno.h>
#include <stdlib.h>

// Inspect only the launched child's descriptors, never a responding port alone.
static int freehand_owns_listener(int port, int pid) {
	int size = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, NULL, 0);
	if (size <= 0) return errno == ESRCH ? 0 : -1;
	if (size > 4 * 1024 * 1024) return -1;
	// Allow descriptors opened between sizing and reading; a full buffer retries
	// on the next readiness poll instead of adopting incomplete metadata.
	int capacity = size + 32 * sizeof(struct proc_fdinfo);
	struct proc_fdinfo *fds = malloc(capacity);
	if (!fds) return -1;
	int bytes = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, fds, capacity);
	if (bytes <= 0 || bytes >= capacity || bytes % sizeof(*fds)) {
		free(fds);
		return bytes < 0 && errno != ESRCH ? -1 : 0;
	}
	int owned = 0;
	for (int i = 0; i < bytes / sizeof(*fds); i++) {
		if (fds[i].proc_fdtype != PROX_FDTYPE_SOCKET) continue;
		struct socket_fdinfo info = {0};
		int n = proc_pidfdinfo(pid, fds[i].proc_fd, PROC_PIDFDSOCKETINFO, &info, sizeof(info));
		if (n != sizeof(info)) continue; // FD closed during the bounded snapshot.
		struct socket_info *s = &info.psi;
		if (s->soi_kind != SOCKINFO_TCP || s->soi_family != AF_INET ||
			s->soi_proto.pri_tcp.tcpsi_state != TSI_S_LISTEN) continue;
		struct in_sockinfo *in = &s->soi_proto.pri_tcp.tcpsi_ini;
		if (ntohs((uint16_t)in->insi_lport) == port &&
			ntohl(in->insi_laddr.ina_46.i46a_addr4.s_addr) == INADDR_LOOPBACK) {
			owned = 1;
			break;
		}
	}
	free(fds);
	return owned;
}
*/
import "C"

import "errors"

func ownsListener(port, pid int) (bool, error) {
	if port < 1 || port > 65535 || pid < 1 {
		return false, errors.New("invalid managed listener identity")
	}
	result := C.freehand_owns_listener(C.int(port), C.int(pid))
	if result < 0 {
		return false, errors.New("could not verify managed listener ownership")
	}
	return result == 1, nil
}
