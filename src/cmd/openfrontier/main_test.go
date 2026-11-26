package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/tiff"

	"github.com/brtrombin/open-frontier/src/internal/imaging"
	"github.com/brtrombin/open-frontier/src/internal/spool"
	"github.com/brtrombin/open-frontier/src/internal/synth"
)

const w, h = 48, 32

func write(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// order writes a two-frame order folder: frame 01A rotated 90, frame 02A not.
func order(t *testing.T, parent string) string {
	dir := filepath.Join(parent, "00006360")
	write(t, filepath.Join(dir, "00000001.RAW"), synth.Raw(synth.Scene(w, h), w, h))
	write(t, filepath.Join(dir, "000001.INF"), synth.Inf(w, h, "01A", 90))
	write(t, filepath.Join(dir, "00000002.RAW"), synth.Raw(synth.Scene(w, h), w, h))
	write(t, filepath.Join(dir, "000002.INF"), synth.Inf(w, h, "02A", 0))
	return dir
}

func runCmd(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func decodeSize(t *testing.T, path string) image.Point {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return image.Point{cfg.Width, cfg.Height}
}

func TestConvert(t *testing.T) {
	root := t.TempDir()
	src := order(t, filepath.Join(root, "copies"))
	out := filepath.Join(root, "out")

	code, stdout, stderr := runCmd("convert", src, "--out", out)
	if code != 0 {
		t.Fatalf("code %d\n%s\n%s", code, stdout, stderr)
	}
	if s := decodeSize(t, filepath.Join(out, "00006360", "00006360-01A.jpg")); s != (image.Point{h, w}) {
		t.Fatalf("01A should be rotated: %v", s)
	}
	if s := decodeSize(t, filepath.Join(out, "00006360", "00006360-02A.jpg")); s != (image.Point{w, h}) {
		t.Fatalf("02A should not be rotated: %v", s)
	}

	code, stdout, _ = runCmd("convert", "--out", out, src)
	if code != 0 || !strings.Contains(stdout, "0 written, 2 already there") {
		t.Fatalf("second run: %d\n%s", code, stdout)
	}

	code, stdout, stderr = runCmd("convert", src, "--out", out, "--format", "tiff", "--tiff-compression", "deflate", "--rotate", "none")
	if code != 0 {
		t.Fatalf("tiff: %d\n%s\n%s", code, stdout, stderr)
	}
	f, err := os.Open(filepath.Join(out, "00006360", "00006360-01A.tif"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := tiff.Decode(f)
	if err != nil || img.Bounds().Dx() != w {
		t.Fatalf("tiff decode: %v %v", err, img)
	}
	leftovers, _ := filepath.Glob(filepath.Join(out, "00006360", ".openfrontier-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left: %v", leftovers)
	}
}

func TestInfInOutSpool(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "copy-in", "00006360")
	write(t, filepath.Join(dir, "00000001.RAW"), synth.Raw(synth.Scene(w, h), w, h))
	write(t, filepath.Join(root, "copy-out", "00006360", "000001.INF"), synth.Inf(w, h, "05A", 90))
	out := filepath.Join(root, "out")
	code, stdout, stderr := runCmd("convert", dir, "--out", out, "--inf-dir", filepath.Join(root, "copy-out"))
	if code != 0 {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	if s := decodeSize(t, filepath.Join(out, "00006360", "00006360-05A.jpg")); s != (image.Point{h, w}) {
		t.Fatalf("size %v", s)
	}
}

func TestCompareFindsOrientation(t *testing.T) {
	root := t.TempDir()
	src := order(t, root)
	raw := filepath.Join(src, "00000001.RAW")
	fr, err := spool.OpenFrame(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := imaging.Orient(imaging.FromFrame(fr, false), "rot90cw")
	var buf bytes.Buffer
	if err := png.Encode(&buf, ref); err != nil {
		t.Fatal(err)
	}
	refPath := write(t, filepath.Join(root, "fuji", "01A.png"), buf.Bytes())

	code, stdout, stderr := runCmd("compare", raw, refPath)
	if code != 0 {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) < 4 || !strings.Contains(lines[3], "rot90cw") || !strings.Contains(lines[3], "rgb") {
		t.Fatalf("best match line wrong:\n%s", stdout)
	}
	for _, want := range []string{"verdict    match", "= 90", "ours 32x48 after rot90cw, reference 32x48"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q:\n%s", want, stdout)
		}
	}
}

func TestInspect(t *testing.T) {
	root := t.TempDir()
	src := order(t, root)
	write(t, filepath.Join(src, "CdOrder.INF"), []byte("Frame 1 0 01A\r\n"))
	code, stdout, stderr := runCmd("inspect", src)
	if code != 0 {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	for _, want := range []string{"ORDER 00006360: 2 frames", "01A", "complete", "other files: CdOrder.INF",
		"layout best  : interleaved 48x32", "02_ROTATION_FOR_PROPER_RECTANGLE"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q:\n%s", want, stdout)
		}
	}
	if code, stdout, _ := runCmd(src); code != 0 || !strings.Contains(stdout, "ORDER 00006360") {
		t.Fatalf("a dropped folder should be inspected: %d\n%s", code, stdout)
	}
	if code, _, _ := runCmd(filepath.Join(src, "000001.INF")); code != 0 {
		t.Fatal("inspect of a .INF failed")
	}
}

func TestSafetyRefusals(t *testing.T) {
	root := t.TempDir()
	live := order(t, filepath.Join(root, "InSpool"))
	if code, _, stderr := runCmd("inspect", live); code != 2 || !strings.Contains(stderr, "wedge the scanner") {
		t.Fatalf("live spool not refused: %d %s", code, stderr)
	}
	if code, _, _ := runCmd("convert", live, "--out", filepath.Join(root, "out")); code != 2 {
		t.Fatal("convert from a live spool not refused")
	}
	if code, stdout, _ := runCmd("inspect", live, "--live-spool-ok"); code != 0 || !strings.Contains(stdout, "warning") {
		t.Fatalf("override failed: %d\n%s", code, stdout)
	}
	src := order(t, filepath.Join(root, "copies"))
	if code, _, stderr := runCmd("convert", src, "--out", filepath.Join(src, "out")); code != 2 || !strings.Contains(stderr, "inside") {
		t.Fatalf("output inside source not refused: %d %s", code, stderr)
	}
	if code, _, _ := runCmd("convert", src, "--out", filepath.Join(root, "OutSpool", "x")); code != 2 {
		t.Fatal("output into a spool-named folder not refused")
	}
}

func TestBadInput(t *testing.T) {
	root := t.TempDir()
	src := order(t, root)
	data, _ := os.ReadFile(filepath.Join(src, "00000002.RAW"))
	write(t, filepath.Join(src, "00000002.RAW"), data[:len(data)-7])
	code, stdout, stderr := runCmd("convert", src, "--out", filepath.Join(root, "out"))
	if code != 1 || !strings.Contains(stderr, "short") || !strings.Contains(stdout, "1 written") {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	for _, args := range [][]string{
		{"convert", src}, {"convert", src, "--out", "x", "--format", "gif"},
		{"compare", src}, {"nonsense-command"}, {"inspect", src, "--bogus"},
	} {
		if code, _, _ := runCmd(args...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}
