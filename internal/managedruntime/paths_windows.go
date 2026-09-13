//go:build windows

package managedruntime

import "golang.org/x/sys/windows"

func isReparse(path string) bool {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return true
	}
	a, e := windows.GetFileAttributes(p)
	return e != nil || a&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
