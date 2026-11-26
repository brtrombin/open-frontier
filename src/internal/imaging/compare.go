package imaging

import (
	"image"
	"sort"
)

// Match is one way of laying our decode over the reference image.
type Match struct {
	Orientation string
	BGR         bool
	Difference  float64 // mean absolute difference per channel, 0 to 255
}

const thumb = 256

// Rank tries every orientation and both channel orders of decoded against
// reference, best first. Both are reduced to the same small square, so crop
// and size differences matter little.
func Rank(decoded, reference *image.RGBA) []Match {
	ref := Downscale(Fit(reference, 4*thumb), thumb, thumb)
	small := Fit(decoded, 4*thumb)
	var out []Match
	for _, bgr := range []bool{false, true} {
		src := small
		if bgr {
			src = swapRedBlue(small)
		}
		for _, name := range Orientations {
			o, _ := Orient(src, name)
			out = append(out, Match{name, bgr, difference(Downscale(o, thumb, thumb), ref)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Difference < out[j].Difference })
	return out
}

func difference(a, b *image.RGBA) float64 {
	total, n := 0, 0
	for i := 0; i+3 < len(a.Pix) && i+3 < len(b.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			d := int(a.Pix[i+c]) - int(b.Pix[i+c])
			if d < 0 {
				d = -d
			}
			total += d
			n++
		}
	}
	if n == 0 {
		return 255
	}
	return float64(total) / float64(n)
}

// Verdict reads a ranking. The thresholds are rules of thumb for 256 x 256
// thumbnails, not measured on scanner files yet.
func Verdict(m []Match) string {
	best := m[0]
	var s string
	switch {
	case best.Difference < 6:
		s = "match: our decode reproduces the reference"
	case best.Difference < 20:
		s = "same picture, but crop, tone or sharpening differ"
	default:
		s = "no clear match: a different frame, or the RAW is not the final rendering"
	}
	if len(m) > 1 && m[1].Difference < 1.5*best.Difference+1 {
		s += " (the runner-up is close, so the orientation is not certain)"
	}
	return s
}
