//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var getConsoleProcessList = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// ownConsole reports whether this process is alone in its console window,
// as when it was started by a double click or a drop in Explorer; the window
// would then close before the output can be read.
func ownConsole() bool {
	var ids [4]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	return n == 1
}
