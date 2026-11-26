//go:build !windows

package spool

import "os"

// OpenShared opens a file for reading. On macOS and Linux the SMB client, not
// this program, chooses the share modes of a network open.
func OpenShared(path string) (*os.File, error) { return os.Open(path) }
