package spool_test

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/brtrombin/open-frontier/src/internal/spool"
	"github.com/brtrombin/open-frontier/src/internal/synth"
)

const w, h = 48, 32

func writeFile(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHeaderAndCandidates(t *testing.T) {
	p := writeFile(t, filepath.Join(t.TempDir(), "00000001.RAW"), synth.Raw(synth.Scene(w, h), w, h))
	hd, err := spool.ReadHeader(p)
	if err != nil {
		t.Fatal(err)
	}
	if !hd.MagicOK() || hd.Dims() != (spool.Size{W: w, H: h}) || hd.FileSize != spool.ExpectedSize(spool.Size{W: w, H: h}) {
		t.Fatalf("header %+v", hd)
	}
	got := spool.Candidates(hd, &spool.Size{W: w, H: h})
	want := []spool.Size{{W: w, H: h}, {W: h, H: w}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates %v, want %v", got, want)
	}
	if spool.Status(hd, nil) != "complete" {
		t.Fatal("status")
	}
}

func TestLayoutDetection(t *testing.T) {
	scene := synth.Scene(w, h)
	cases := []struct {
		name           string
		pixels         []byte
		headerW, headH int
		planar         bool
	}{
		{"interleaved", scene, w, h, false},
		{"planar", synth.Planar(scene), w, h, true},
		{"header with swapped dimensions", scene, h, w, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeFile(t, filepath.Join(t.TempDir(), "1.RAW"), synth.Raw(c.pixels, c.headerW, c.headH))
			fr, err := spool.OpenFrame(p, nil)
			if err != nil {
				t.Fatal(err)
			}
			best := fr.Best()
			if best.Size != (spool.Size{W: w, H: h}) || best.Planar != c.planar || fr.Ambiguous() {
				t.Fatalf("best %v, ambiguous %v, all %v", best, fr.Ambiguous(), fr.Layouts)
			}
		})
	}
}

func TestShortFrame(t *testing.T) {
	data := synth.Raw(synth.Scene(w, h), w, h)
	p := writeFile(t, filepath.Join(t.TempDir(), "1.RAW"), data[:len(data)-100])
	_, err := spool.OpenFrame(p, nil)
	if !errors.Is(err, spool.ErrIncomplete) || !strings.Contains(err.Error(), "short") {
		t.Fatalf("got %v", err)
	}
	hd, _ := spool.ReadHeader(p)
	if spool.Status(hd, nil) != "SHORT" {
		t.Fatal("status should be SHORT")
	}
}

func TestParseInf(t *testing.T) {
	data := synth.Inf(w, h, "01A", 90)
	data = append(data, []byte("junk")...)
	data = append(data, synth.Record("02_DX", 0x0013, binary.LittleEndian.AppendUint32(nil, 5))...)
	data = append(data, synth.Array("02_SCAN_SIZE", 0x0013, 10, 20)...)
	rec := synth.U32("02_MASKKIND", 0)
	padded := append(append([]byte{}, rec[:12]...), 0) // a second NUL after the key
	data = append(append(data, padded...), rec[12:]...)
	data = append(data, []byte("99_UNKNOWN_KEY\x00\x10\x00\x77\x77abcd")...)

	inf := spool.ParseInf(data)
	if s := inf.ImageSize(); s == nil || *s != (spool.Size{W: w, H: h}) {
		t.Fatalf("image size %v", s)
	}
	if r, ok := inf.Rotation(); !ok || r != 90 {
		t.Fatalf("rotation %v %v", r, ok)
	}
	if inf.FrameName() != "01A" || inf["00_MODEL_NAME"].Value != "SP-3000" {
		t.Fatalf("strings %q %v", inf.FrameName(), inf["00_MODEL_NAME"].Value)
	}
	if d, ok := inf["02_SCANNING_DATE"].Value.(time.Time); !ok || !d.Equal(synth.ScanDate) {
		t.Fatalf("scanning date %v", inf["02_SCANNING_DATE"].Value)
	}
	if inf["02_DX"].Value != uint32(5) {
		t.Fatalf("0x0013 scalar: %v", inf["02_DX"].Value)
	}
	if !reflect.DeepEqual(inf["02_SCAN_SIZE"].Value, []uint32{10, 20}) {
		t.Fatalf("0x0013 array: %v", inf["02_SCAN_SIZE"].Value)
	}
	if inf["02_MASKKIND"].Value != uint32(0) || !inf["02_MASKKIND"].Known {
		t.Fatalf("padded record: %+v", inf["02_MASKKIND"])
	}
	if f, ok := inf["99_UNKNOWN_KEY"]; !ok || f.Known || f.Value != nil {
		t.Fatalf("unknown key: %+v", f)
	}
	for _, f := range inf.Sorted() {
		if spool.FormatValue(f) == "" {
			t.Fatalf("empty rendering for %s", f.Key)
		}
	}
	for n := range data {
		spool.ParseInf(data[:n])
	}
}

func TestFindFrames(t *testing.T) {
	root := t.TempDir()
	order := filepath.Join(root, "orders", "00006360")
	writeFile(t, filepath.Join(order, "000001.RAW"), []byte("x"))
	writeFile(t, filepath.Join(order, "000001.INF"), []byte("x"))
	writeFile(t, filepath.Join(order, "00000002.raw"), []byte("x"))
	writeFile(t, filepath.Join(order, "00000003.RAW"), []byte("x"))
	writeFile(t, filepath.Join(order, "._00000003.RAW"), []byte("x"))
	writeFile(t, filepath.Join(order, "CdOrder.INF"), []byte("x"))
	out := filepath.Join(root, "OutSpool")
	writeFile(t, filepath.Join(out, "00006360", "000002.INF"), []byte("x"))
	writeFile(t, filepath.Join(out, "000003.INF"), []byte("x"))

	pairs, err := spool.FindFrames(order, out)
	if err != nil {
		t.Fatal(err)
	}
	want := []spool.FramePair{
		{Raw: filepath.Join(order, "000001.RAW"), Inf: filepath.Join(order, "000001.INF")},
		{Raw: filepath.Join(order, "00000002.raw"), Inf: filepath.Join(out, "00006360", "000002.INF")},
		{Raw: filepath.Join(order, "00000003.RAW"), Inf: filepath.Join(out, "000003.INF")},
	}
	if !reflect.DeepEqual(pairs, want) {
		t.Fatalf("got %v\nwant %v", pairs, want)
	}
	if pairs, _ := spool.FindFrames(order, ""); pairs[1].Inf != "" {
		t.Fatal("no inf dir: frame 2 should have no .INF")
	}
	others, _ := spool.OtherFiles(order)
	if !reflect.DeepEqual(others, []string{"CdOrder.INF"}) {
		t.Fatalf("other files %v", others)
	}
	dirs, _ := spool.OrderDirs(filepath.Join(root, "orders"))
	if !reflect.DeepEqual(dirs, []string{order}) {
		t.Fatalf("order dirs %v", dirs)
	}
}

func TestSpoolNames(t *testing.T) {
	root := t.TempDir()
	if r := spool.SpoolNameReason(filepath.Join(root, "InSpool", "00006360")); !strings.Contains(r, "InSpool") {
		t.Fatalf("got %q", r)
	}
	if r := spool.LiveSpoolReason(filepath.Join(root, "copies", "00006360")); r != "" {
		t.Fatalf("local copy flagged: %q", r)
	}
}

func TestMountTables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX paths")
	}
	linux := spool.ParseProcMounts("/dev/sda1 / ext4 rw 0 0\n//ezpc00/InSpool /mnt/scan\\040pc cifs rw 0 0\n")
	if !reflect.DeepEqual(linux, []spool.Mount{{Point: "/", Type: "ext4"}, {Point: "/mnt/scan pc", Type: "cifs"}}) {
		t.Fatalf("linux %v", linux)
	}
	mac := spool.ParseBSDMount("/dev/disk3s1s1 on / (apfs, sealed, local, read-only, journaled)\n" +
		"//FRONTIER@EZPC00/InSpool on /Volumes/In Spool (smbfs, nodev, nosuid, mounted by me)\n")
	if !reflect.DeepEqual(mac, []spool.Mount{{Point: "/", Type: "apfs"}, {Point: "/Volumes/In Spool", Type: "smbfs"}}) {
		t.Fatalf("mac %v", mac)
	}
	if got := spool.FilesystemType("/Volumes/In Spool/00006360", mac); got != "smbfs" {
		t.Fatalf("longest prefix: %q", got)
	}
	if got := spool.FilesystemType("/Volumes/In Spoolx", mac); got != "apfs" {
		t.Fatalf("prefix must stop at a separator: %q", got)
	}
}
