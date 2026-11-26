package spool

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var spoolNames = map[string]bool{"inspool": true, "outspool": true, "imgexchange": true}

// LiveSpoolReason says why path may be a folder the scanner still writes
// into, or returns "" when it looks like a local copy. Reading a live spool
// while the scanner owns the files can abort ExecLSeqObj.exe and wedge the
// scanner (docs/protocol.md, section 9.9).
func LiveSpoolReason(path string) string {
	if r := SpoolNameReason(path); r != "" {
		return r
	}
	return networkReason(path)
}

// SpoolNameReason flags paths inside a folder named like a Fuji spool folder.
func SpoolNameReason(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	for _, part := range strings.FieldsFunc(abs, func(r rune) bool { return r == '/' || r == '\\' }) {
		if spoolNames[strings.ToLower(part)] {
			return "it is inside a folder named " + part + ", like the scanner's spool"
		}
	}
	return ""
}

// Mount is one entry of a mount table.
type Mount struct{ Point, Type string }

var octalEscape = regexp.MustCompile(`\\[0-7]{3}`)

// ParseProcMounts reads Linux /proc/self/mounts.
func ParseProcMounts(text string) []Mount {
	var out []Mount
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		point := octalEscape.ReplaceAllStringFunc(f[1], func(s string) string {
			n, _ := strconv.ParseUint(s[1:], 8, 8)
			return string(rune(n))
		})
		out = append(out, Mount{point, f[2]})
	}
	return out
}

// ParseBSDMount reads the output of the macOS mount command, with lines like
// "//user@host/share on /Volumes/share (smbfs, nodev, nosuid, mounted by user)".
func ParseBSDMount(text string) []Mount {
	var out []Mount
	for _, line := range strings.Split(text, "\n") {
		_, rest, ok := strings.Cut(line, " on ")
		if !ok {
			continue
		}
		i := strings.LastIndex(rest, " (")
		if i < 0 {
			continue
		}
		typ, _, _ := strings.Cut(strings.TrimSuffix(rest[i+2:], ")"), ",")
		out = append(out, Mount{rest[:i], strings.TrimSpace(typ)})
	}
	return out
}

// FilesystemType is the type of the longest mount point that contains path.
func FilesystemType(path string, table []Mount) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	best, typ := -1, ""
	for _, m := range table {
		p := strings.TrimSuffix(m.Point, "/")
		if (p == "" || abs == p || strings.HasPrefix(abs, p+"/")) && len(p) > best {
			best, typ = len(p), m.Type
		}
	}
	return typ
}
