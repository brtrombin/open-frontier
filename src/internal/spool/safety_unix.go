//go:build !windows

package spool

import (
	"os"
	"os/exec"
)

var networkTypes = map[string]bool{
	"smbfs": true, "cifs": true, "smb3": true, "nfs": true, "nfs4": true,
	"afpfs": true, "webdav": true, "fuse.sshfs": true,
}

func networkReason(path string) string {
	if t := FilesystemType(path, mountTable()); networkTypes[t] {
		return "it is on a network share (" + t + ")"
	}
	return ""
}

func mountTable() []Mount {
	if data, err := os.ReadFile("/proc/self/mounts"); err == nil {
		return ParseProcMounts(string(data))
	}
	out, err := exec.Command("/sbin/mount").Output()
	if err != nil {
		return nil
	}
	return ParseBSDMount(string(out))
}
