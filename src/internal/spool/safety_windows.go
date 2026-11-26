//go:build windows

package spool

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var getDriveType = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDriveTypeW")

const driveRemote = 4

func networkReason(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	vol := filepath.VolumeName(abs)
	if strings.HasPrefix(vol, `\\`) {
		return "it is a network path (" + vol + ")"
	}
	if len(vol) != 2 {
		return ""
	}
	root, err := syscall.UTF16PtrFromString(vol + `\`)
	if err != nil {
		return ""
	}
	if t, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(root))); t == driveRemote {
		return "it is on a network drive (" + vol + ")"
	}
	return ""
}
