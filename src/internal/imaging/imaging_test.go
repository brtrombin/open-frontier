package imaging

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/tiff"

	"github.com/brtrombin/open-frontier/src/internal/spool"
	"github.com/brtrombin/open-frontier/src/internal/synth"
)

// labelled is a 3 x 2 image whose red values are 1 to 6, row by row:
//
//	1 2 3
//	4 5 6
func labelled() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for i := 0; i < 6; i++ {
		img.Pix[4*i], img.Pix[4*i+3] = byte(i+1), 255
	}
	return img
}

func reds(img *image.RGBA) []byte {
	var out []byte
	for i := 0; i < len(img.Pix); i += 4 {
		out = append(out, img.Pix[i])
	}
	return out
}

func TestOrient(t *testing.T) {
	want := map[string]struct {
		w, h int
		red  string
	}{
		"as-is":           {3, 2, "\x01\x02\x03\x04\x05\x06"},
		"rot90cw":         {2, 3, "\x04\x01\x05\x02\x06\x03"},
		"rot180":          {3, 2, "\x06\x05\x04\x03\x02\x01"},
		"rot90ccw":        {2, 3, "\x03\x06\x02\x05\x01\x04"},
		"mirror":          {3, 2, "\x03\x02\x01\x06\x05\x04"},
		"mirror+rot90cw":  {2, 3, "\x06\x03\x05\x02\x04\x01"},
		"mirror+rot180":   {3, 2, "\x04\x05\x06\x01\x02\x03"},
		"mirror+rot90ccw": {2, 3, "\x01\x04\x02\x05\x03\x06"},
	}
	if len(want) != len(Orientations) {
		t.Fatal("every orientation needs a case")
	}
	for _, name := range Orientations {
		got, err := Orient(labelled(), name)
		if err != nil {
			t.Fatal(err)
		}
		w := want[name]
		if got.Rect.Dx() != w.w || got.Rect.Dy() != w.h || string(reds(got)) != w.red {
			t.Errorf("%s: %dx%d %v", name, got.Rect.Dx(), got.Rect.Dy(), reds(got))
		}
	}
	if _, err := Orient(labelled(), "sideways"); err == nil {
		t.Fatal("unknown orientation accepted")
	}
}

func TestDownscale(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for i, v := range []byte{0, 10, 100, 200, 20, 30, 100, 0} {
		img.Pix[4*i] = v
	}
	got := reds(Downscale(img, 2, 1))
	if string(got) != string([]byte{15, 100}) {
		t.Fatalf("got %v", got)
	}
	if f := Fit(image.NewRGBA(image.Rect(0, 0, 4000, 1000)), 1024); f.Rect.Dx() != 1024 || f.Rect.Dy() != 256 {
		t.Fatalf("fit %v", f.Rect)
	}
}

func frame(t *testing.T, w, h int, planar bool) *spool.Frame {
	t.Helper()
	px := synth.Scene(w, h)
	if planar {
		px = synth.Planar(px)
	}
	p := filepath.Join(t.TempDir(), "1.RAW")
	if err := os.WriteFile(p, synth.Raw(px, w, h), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := spool.OpenFrame(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFromFrame(t *testing.T) {
	for _, planar := range []bool{false, true} {
		f := frame(t, 48, 32, planar)
		rgb := FromFrame(f, false)
		if got := rgb.Pix[:4]; got[0] != 0 || got[1] != 0 || got[2] != 250 {
			t.Fatalf("planar=%v first pixel %v", planar, got)
		}
		if last := rgb.Pix[len(rgb.Pix)-4:]; last[0] != 255 || last[1] != 255 || last[2] != 30 {
			t.Fatalf("planar=%v last pixel %v", planar, last)
		}
		if bgr := FromFrame(f, true).Pix[:3]; bgr[0] != 250 || bgr[2] != 0 {
			t.Fatalf("bgr %v", bgr)
		}
	}
}

func TestTIFFRoundTrip(t *testing.T) {
	img := FromFrame(frame(t, 48, 70, false), false)
	for _, deflate := range []bool{false, true} {
		var buf bytes.Buffer
		if err := WriteTIFF(&buf, img, deflate); err != nil {
			t.Fatal(err)
		}
		dec, err := tiff.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("deflate=%v: %v", deflate, err)
		}
		b := dec.Bounds()
		if b.Dx() != 48 || b.Dy() != 70 {
			t.Fatalf("size %v", b)
		}
		for y := 0; y < 70; y++ {
			for x := 0; x < 48; x++ {
				r, g, bl, _ := dec.At(x, y).RGBA()
				o := img.PixOffset(x, y)
				if byte(r>>8) != img.Pix[o] || byte(g>>8) != img.Pix[o+1] || byte(bl>>8) != img.Pix[o+2] {
					t.Fatalf("deflate=%v pixel %d,%d differs", deflate, x, y)
				}
			}
		}
	}
}

func exifJPEG(order binary.AppendByteOrder, mark string, value uint16) []byte {
	t := []byte(mark)
	t = order.AppendUint16(t, 42)
	t = order.AppendUint32(t, 8)
	t = order.AppendUint16(t, 1)
	t = order.AppendUint16(t, 0x0112)
	t = order.AppendUint16(t, 3)
	t = order.AppendUint32(t, 1)
	t = order.AppendUint16(t, value)
	t = append(t, 0, 0, 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), t...)
	out := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 4, 0, 0, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	return append(append(out, seg...), 0xFF, 0xDA, 0, 2)
}

func TestExifOrientation(t *testing.T) {
	if got := ExifOrientation(exifJPEG(binary.LittleEndian, "II", 6)); got != "rot90cw" {
		t.Fatalf("II: %q", got)
	}
	if got := ExifOrientation(exifJPEG(binary.BigEndian, "MM", 8)); got != "rot90ccw" {
		t.Fatalf("MM: %q", got)
	}
	if got := ExifOrientation([]byte{0xFF, 0xD8, 0xFF, 0xDA, 0, 2}); got != "" {
		t.Fatalf("no EXIF: %q", got)
	}
	data := exifJPEG(binary.LittleEndian, "II", 3)
	for n := range data {
		ExifOrientation(data[:n])
	}
}

func TestRank(t *testing.T) {
	decoded := FromFrame(frame(t, 48, 32, false), false)
	ref, err := Orient(swapRedBlue(decoded), "mirror+rot90cw")
	if err != nil {
		t.Fatal(err)
	}
	m := Rank(decoded, ref)
	if m[0].Orientation != "mirror+rot90cw" || !m[0].BGR || m[0].Difference > 0.5 {
		t.Fatalf("best %+v", m[0])
	}
	if m[1].Difference < 5 {
		t.Fatalf("runner-up too close: %+v", m[1])
	}
	if v := Verdict(m); v[:5] != "match" {
		t.Fatalf("verdict %q", v)
	}
}
