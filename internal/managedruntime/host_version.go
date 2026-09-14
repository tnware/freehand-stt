package managedruntime

import (
	"strconv"
	"strings"
)

func macOSVersion(value string) uint32 {
	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return 0
	}
	var version uint32
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return 0
		}
		version |= uint32(n) << (16 - 8*i)
	}
	return version
}

func supportsOSVersion(provider ProviderID, os string, version uint32) bool {
	// The official b10809 macOS server and its dylibs target Ventura 13.3.
	// This is independent of Freehand's macOS 13.0 application minimum.
	return os != "darwin" || provider != LlamaCPP || version >= macOSVersion("13.3")
}
