// Package imaging turns spool frames into oriented images, writes them as
// JPEG or TIFF, and compares them with the Fuji software's own output.
package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"github.com/brtrombin/open-frontier/src/internal/spool"
)

// Orientations are the eight ways to lay a frame down. "mirror" flips left
// to right before the rotation.
var Orientations = []string{
	"as-is", "rot90cw", "rot180", "rot90ccw",
	"mirror", "mirror+rot90cw", "mirror+rot180", "mirror+rot90ccw",
}

// FromFrame builds an image from the frame's best layout; bgr swaps red and blue.
func FromFrame(f *spool.Frame, bgr bool) *image.RGBA {
	l := f.Best()
	img := image.NewRGBA(image.Rect(0, 0, l.W, l.H))
	px := f.Pixels()
	n := l.W * l.H
	ri, bi := 0, 2
	if bgr {
		ri, bi = 2, 0
	}
	var c [3]byte
	for i := 0; i < n; i++ {
		if l.Planar {
			c = [3]byte{px[i], px[n+i], px[2*n+i]}
		} else {
			c = [3]byte{px[3*i], px[3*i+1], px[3*i+2]}
		}
		o := 4 * i
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c[ri], c[1], c[bi], 255
	}
	return img
}

// Orient returns a copy of img, mirrored and then rotated as name says.
func Orient(img *image.RGBA, name string) (*image.RGBA, error) {
	mirror := strings.HasPrefix(name, "mirror")
	rot := strings.TrimPrefix(strings.TrimPrefix(name, "mirror"), "+")
	if rot == "" {
		rot = "as-is"
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	dw, dh := w, h
	var to func(x, y int) (int, int)
	switch rot {
	case "as-is":
		to = func(x, y int) (int, int) { return x, y }
	case "rot90cw":
		dw, dh = h, w
		to = func(x, y int) (int, int) { return h - 1 - y, x }
	case "rot180":
		to = func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }
	case "rot90ccw":
		dw, dh = h, w
		to = func(x, y int) (int, int) { return y, w - 1 - x }
	default:
		return nil, fmt.Errorf("unknown orientation %q", name)
	}
	out := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			sx := x
			if mirror {
				sx = w - 1 - x
			}
			dx, dy := to(x, y)
			o := dy*out.Stride + 4*dx
			copy(out.Pix[o:o+4], row[4*sx:4*sx+4])
		}
	}
	return out, nil
}

// Downscale averages img into a w x h image.
func Downscale(img *image.RGBA, w, h int) *image.RGBA {
	sw, sh := img.Rect.Dx(), img.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for dy := 0; dy < h; dy++ {
		y0, y1 := span(dy, h, sh)
		for dx := 0; dx < w; dx++ {
			x0, x1 := span(dx, w, sw)
			var r, g, b, n int
			for y := y0; y < y1; y++ {
				row := img.Pix[y*img.Stride:]
				for x := x0; x < x1; x++ {
					r += int(row[4*x])
					g += int(row[4*x+1])
					b += int(row[4*x+2])
					n++
				}
			}
			o := dy*out.Stride + 4*dx
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return out
}

func span(i, n, size int) (int, int) {
	a, b := i*size/n, (i+1)*size/n
	if b <= a {
		b = a + 1
	}
	return a, b
}

// Fit shrinks img so that neither side exceeds limit, keeping its shape.
func Fit(img *image.RGBA, limit int) *image.RGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	switch {
	case w <= limit && h <= limit:
		return img
	case w >= h:
		return Downscale(img, limit, maxInt(1, limit*h/w))
	default:
		return Downscale(img, maxInt(1, limit*w/h), limit)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func swapRedBlue(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	copy(out.Pix, img.Pix)
	for i := 0; i+3 < len(out.Pix); i += 4 {
		out.Pix[i], out.Pix[i+2] = out.Pix[i+2], out.Pix[i]
	}
	return out
}

// LoadReference reads a JPEG, PNG, TIFF or BMP image and applies its EXIF
// orientation, returning the orientation it applied ("" when none).
func LoadReference(path string) (*image.RGBA, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("%s: %v", path, err)
	}
	b := src.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Rect, src, b.Min, draw.Src)
	o := ExifOrientation(data)
	if o == "" || o == "as-is" {
		return img, "", nil
	}
	img, err = Orient(img, o)
	return img, o, err
}
