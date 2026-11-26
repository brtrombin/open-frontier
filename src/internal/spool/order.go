package spool

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// FramePair is a .RAW file and the .INF that describes it ("" when none was found).
type FramePair struct {
	Raw, Inf string
}

// FindFrames pairs each .RAW file of an order folder with its .INF, looking
// beside it first, then in infDir/<order folder name>, then in infDir itself.
func FindFrames(orderDir, infDir string) ([]FramePair, error) {
	raws, err := listFiles(orderDir, ".raw")
	if err != nil {
		return nil, err
	}
	folders := []string{orderDir}
	if infDir != "" {
		folders = append(folders, filepath.Join(infDir, filepath.Base(orderDir)), infDir)
	}
	infs := map[string]string{}
	for _, dir := range folders {
		files, err := listFiles(dir, ".inf")
		if err != nil {
			continue
		}
		for _, f := range files {
			if _, ok := infs[frameKey(f)]; !ok {
				infs[frameKey(f)] = f
			}
		}
	}
	sort.SliceStable(raws, func(i, j int) bool { return frameLess(raws[i], raws[j]) })
	pairs := make([]FramePair, 0, len(raws))
	for _, r := range raws {
		pairs = append(pairs, FramePair{r, infs[frameKey(r)]})
	}
	return pairs, nil
}

func frameNumber(path string) (uint64, bool) {
	n, err := strconv.ParseUint(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), 10, 64)
	return n, err == nil
}

// frameKey makes 000007.INF pair with 000007.RAW and with 00000007.RAW.
func frameKey(path string) string {
	if n, ok := frameNumber(path); ok {
		return "#" + strconv.FormatUint(n, 10)
	}
	return strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

// frameLess orders numbered frames by number, before any other names.
func frameLess(a, b string) bool {
	na, okA := frameNumber(a)
	nb, okB := frameNumber(b)
	switch {
	case okA && okB && na != nb:
		return na < nb
	case okA != okB:
		return okA
	}
	return filepath.Base(a) < filepath.Base(b)
}

// OrderDirs is the folder itself when it holds .RAW files, else its
// subfolders that do.
func OrderDirs(path string) ([]string, error) {
	raws, err := listFiles(path, ".raw")
	if err != nil {
		return nil, err
	}
	if len(raws) > 0 {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		d := filepath.Join(path, e.Name())
		if r, err := listFiles(d, ".raw"); err == nil && len(r) > 0 {
			dirs = append(dirs, d)
		}
	}
	return dirs, nil
}

// OtherFiles names the files of an order folder that are neither frames nor
// frame .INFs, such as CdOrder.INF.
func OtherFiles(orderDir string) ([]string, error) {
	entries, err := os.ReadDir(orderDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if e.IsDir() || ext == ".raw" || (ext == ".inf" && strings.HasPrefix(frameKey(name), "#")) {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

func listFiles(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ext) {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out, nil
}
