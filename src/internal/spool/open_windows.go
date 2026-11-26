//go:build windows

package spool

import (
	"os"
	"syscall"
)

// OpenShared opens a file for reading while letting the Fuji software keep
// writing, renaming and deleting it. A read that withholds FILE_SHARE_DELETE
// can abort ExecLSeqObj.exe and wedge the scanner.
func OpenShared(path string) (*os.File, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}
