// Package spool reads the frame files the scanner PC of a Fuji Frontier
// SP-3000 writes into its spool folders. Format notes: docs/protocol.md,
// section 9.
package spool

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
)

// HeaderSize is the length of the header in front of the pixels of a .RAW file.
const HeaderSize = 32

// ErrIncomplete means a .RAW file size fits none of its candidate dimensions.
var ErrIncomplete = errors.New("frame size does not fit its dimensions")

// Size is a frame size in pixels.
type Size struct{ W, H int }

// Swap exchanges width and height.
func (s Size) Swap() Size { return Size{s.H, s.W} }

func (s Size) String() string { return fmt.Sprintf("%dx%d", s.W, s.H) }

// ExpectedSize is the byte size of a complete .RAW file of this size.
func ExpectedSize(s Size) int64 { return HeaderSize + int64(s.W)*int64(s.H)*3 }

// Header holds the 16 little-endian words at the start of a .RAW file.
type Header struct {
	Words    [16]uint16
	FileSize int64
}

// MagicOK reports whether the file starts with "DR".
func (h Header) MagicOK() bool { return h.Words[0] == 'D'|'R'<<8 }

// Magic is the first two bytes of the file.
func (h Header) Magic() string { return string([]byte{byte(h.Words[0]), byte(h.Words[0] >> 8)}) }

// Dims is (width, height) as read from header words 4 and 5.
func (h Header) Dims() Size { return Size{int(h.Words[4]), int(h.Words[5])} }

// ReadHeader reads the header of a .RAW file and its size on disk.
func ReadHeader(path string) (Header, error) {
	f, err := OpenShared(path)
	if err != nil {
		return Header{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Header{}, err
	}
	var buf [HeaderSize]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return Header{}, fmt.Errorf("%s: shorter than the %d-byte header", path, HeaderSize)
	}
	h := Header{FileSize: info.Size()}
	for i := range h.Words {
		h.Words[i] = binary.LittleEndian.Uint16(buf[2*i:])
	}
	return h, nil
}

// Candidates lists the (width, height) pairs from the header and from the
// .INF whose pixel count fills the file exactly.
func Candidates(h Header, inf *Size) []Size {
	pairs := []Size{h.Dims(), h.Dims().Swap()}
	if inf != nil {
		pairs = append(pairs, *inf, inf.Swap())
	}
	var out []Size
	for _, s := range pairs {
		if s.W > 0 && s.H > 0 && ExpectedSize(s) == h.FileSize && !containsSize(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func containsSize(list []Size, s Size) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Status is "complete", "SHORT" or "SIZE?", for frame tables.
func Status(h Header, inf *Size) string {
	if len(Candidates(h, inf)) > 0 {
		return "complete"
	}
	if h.FileSize < ExpectedSize(declared(h, inf)) {
		return "SHORT"
	}
	return "SIZE?"
}

// SizeProblem explains why a file fits none of its candidate dimensions.
func SizeProblem(h Header, inf *Size) string {
	s := declared(h, inf)
	want := ExpectedSize(s)
	if h.FileSize < want {
		return fmt.Sprintf("short: %d bytes, a complete %s frame has %d (still being written, or truncated)", h.FileSize, s, want)
	}
	return fmt.Sprintf("%d bytes does not fit %s (%d bytes expected): the header or the format differ from what we know", h.FileSize, s, want)
}

func declared(h Header, inf *Size) Size {
	if inf != nil {
		return *inf
	}
	return h.Dims()
}

// Layout is one way to read the pixels of a frame.
type Layout struct {
	Size
	Planar bool    // R, G and B in three planes instead of interleaved
	Score  float64 // mean difference between samples one image row apart
}

func (l Layout) String() string {
	if l.Planar {
		return "planar " + l.Size.String()
	}
	return "interleaved " + l.Size.String()
}

// RankLayouts orders the possible layouts best first: the true row length
// gives the smallest difference between samples one image row apart.
func RankLayouts(pixels []byte, dims []Size) []Layout {
	var out []Layout
	for _, s := range dims {
		out = append(out,
			Layout{s, false, rowDifference(pixels, 3*s.W)},
			Layout{s, true, rowDifference(pixels, s.W)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score < out[j].Score })
	return out
}

func rowDifference(p []byte, row int) float64 {
	const samples = 20000
	span := len(p) - row
	if span <= 0 {
		return math.Inf(1)
	}
	rng := rand.New(rand.NewSource(1))
	total := 0
	for i := 0; i < samples; i++ {
		k := rng.Intn(span)
		d := int(p[k]) - int(p[k+row])
		if d < 0 {
			d = -d
		}
		total += d
	}
	return float64(total) / samples
}

// Frame is a complete .RAW file in memory.
type Frame struct {
	Path    string
	Header  Header
	Data    []byte
	Layouts []Layout
}

// Pixels is the data after the header.
func (f *Frame) Pixels() []byte { return f.Data[HeaderSize:] }

// Best is the most likely layout.
func (f *Frame) Best() Layout { return f.Layouts[0] }

// Ambiguous reports whether the best layout barely beats the next one, as on
// a flat or blank frame.
func (f *Frame) Ambiguous() bool {
	return len(f.Layouts) > 1 && f.Layouts[1].Score <= 1.2*f.Layouts[0].Score
}

// OpenFrame reads a complete .RAW file, using the .INF size when known.
func OpenFrame(path string, inf *Size) (*Frame, error) {
	h, err := ReadHeader(path)
	if err != nil {
		return nil, err
	}
	dims := Candidates(h, inf)
	if len(dims) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrIncomplete, SizeProblem(h, inf))
	}
	data, err := ReadFileShared(path)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != h.FileSize {
		return nil, fmt.Errorf("%s changed size while it was read", path)
	}
	return &Frame{Path: path, Header: h, Data: data, Layouts: RankLayouts(data[HeaderSize:], dims)}, nil
}

// ReadFileShared reads a whole file through OpenShared.
func ReadFileShared(path string) ([]byte, error) {
	f, err := OpenShared(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	data := make([]byte, info.Size())
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return data, nil
}
