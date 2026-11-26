package imaging

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"io"
)

// WriteTIFF writes img as an 8-bit RGB baseline TIFF, uncompressed or with
// Deflate compression and a horizontal predictor.
func WriteTIFF(w io.Writer, img *image.RGBA, deflate bool) error {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	const rowsPerStrip = 32
	var strips [][]byte
	for y0 := 0; y0 < height; y0 += rowsPerStrip {
		y1 := y0 + rowsPerStrip
		if y1 > height {
			y1 = height
		}
		raw := make([]byte, 0, (y1-y0)*width*3)
		for y := y0; y < y1; y++ {
			row := img.Pix[y*img.Stride : y*img.Stride+4*width]
			start := len(raw)
			for x := 0; x < width; x++ {
				raw = append(raw, row[4*x], row[4*x+1], row[4*x+2])
			}
			if deflate {
				r := raw[start:]
				for i := len(r) - 1; i >= 3; i-- {
					r[i] -= r[i-3]
				}
			}
		}
		if deflate {
			var buf bytes.Buffer
			zw, _ := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
			if _, err := zw.Write(raw); err != nil {
				return err
			}
			if err := zw.Close(); err != nil {
				return err
			}
			raw = buf.Bytes()
		}
		strips = append(strips, raw)
	}

	offsets := make([]uint32, len(strips))
	counts := make([]uint32, len(strips))
	pos := uint32(8)
	for i, s := range strips {
		offsets[i], counts[i] = pos, uint32(len(s))
		pos += uint32(len(s))
	}
	pad := pos % 2 // the IFD starts on a word boundary
	ifdOffset := pos + pad

	compression, predictor := uint16(1), []tiffEntry(nil)
	if deflate {
		compression = 8
		predictor = []tiffEntry{{317, 3, 1, shorts(2)}}
	}
	entries := append([]tiffEntry{
		{256, 4, 1, longs(uint32(width))},
		{257, 4, 1, longs(uint32(height))},
		{258, 3, 3, shorts(8, 8, 8)},
		{259, 3, 1, shorts(compression)},
		{262, 3, 1, shorts(2)},
		{273, 4, uint32(len(offsets)), longs(offsets...)},
		{277, 3, 1, shorts(3)},
		{278, 4, 1, longs(rowsPerStrip)},
		{279, 4, uint32(len(counts)), longs(counts...)},
		{282, 5, 1, longs(72, 1)},
		{283, 5, 1, longs(72, 1)},
		{284, 3, 1, shorts(1)},
		{296, 3, 1, shorts(2)},
	}, predictor...)

	var ifd, extra bytes.Buffer
	le := binary.LittleEndian
	extraOffset := ifdOffset + uint32(2+12*len(entries)+4)
	ifd.Write(le.AppendUint16(nil, uint16(len(entries))))
	for _, e := range entries {
		ifd.Write(le.AppendUint16(nil, e.tag))
		ifd.Write(le.AppendUint16(nil, e.typ))
		ifd.Write(le.AppendUint32(nil, e.count))
		if len(e.value) <= 4 {
			v := make([]byte, 4)
			copy(v, e.value)
			ifd.Write(v)
			continue
		}
		ifd.Write(le.AppendUint32(nil, extraOffset+uint32(extra.Len())))
		extra.Write(e.value)
	}
	ifd.Write(le.AppendUint32(nil, 0))

	header := []byte{'I', 'I', 42, 0, 0, 0, 0, 0}
	le.PutUint32(header[4:], ifdOffset)
	ew := &errWriter{w: w}
	ew.write(header)
	for _, s := range strips {
		ew.write(s)
	}
	ew.write(make([]byte, pad))
	ew.write(ifd.Bytes())
	ew.write(extra.Bytes())
	return ew.err
}

type tiffEntry struct {
	tag, typ uint16
	count    uint32
	value    []byte
}

func shorts(v ...uint16) []byte {
	var b []byte
	for _, x := range v {
		b = binary.LittleEndian.AppendUint16(b, x)
	}
	return b
}

func longs(v ...uint32) []byte {
	var b []byte
	for _, x := range v {
		b = binary.LittleEndian.AppendUint32(b, x)
	}
	return b
}

type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) write(b []byte) {
	if e.err == nil && len(b) > 0 {
		_, e.err = e.w.Write(b)
	}
}
