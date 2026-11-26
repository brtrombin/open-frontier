// Package synth builds synthetic spool files, for the tests and for the
// simulate command. They follow our own reading of the formats
// (docs/protocol.md), not real scanner files.
package synth

import (
	"encoding/binary"
	"time"
	"unicode/utf16"
)

// Scene is an interleaved RGB picture with no symmetry: red grows to the
// right, green grows downwards and a blue patch sits at the top left.
func Scene(w, h int) []byte {
	p := make([]byte, 0, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b := byte(30)
			if x < w/4 && y < h/3 {
				b = 250
			}
			p = append(p, byte(255*x/(w-1)), byte(255*y/(h-1)), b)
		}
	}
	return p
}

// Planar rearranges interleaved RGB into three planes.
func Planar(rgb []byte) []byte {
	n := len(rgb) / 3
	out := make([]byte, len(rgb))
	for i := 0; i < n; i++ {
		out[i], out[n+i], out[2*n+i] = rgb[3*i], rgb[3*i+1], rgb[3*i+2]
	}
	return out
}

// Raw puts a .RAW header declaring w x h in words 4 and 5 in front of pixels.
func Raw(pixels []byte, w, h int) []byte {
	head := make([]byte, 32)
	head[0], head[1] = 'D', 'R'
	binary.LittleEndian.PutUint16(head[8:], uint16(w))
	binary.LittleEndian.PutUint16(head[10:], uint16(h))
	return append(head, pixels...)
}

// Record is one DIPP record: key, NUL, record size, type code, payload.
func Record(key string, typ uint16, payload []byte) []byte {
	b := append([]byte(key), 0)
	b = binary.LittleEndian.AppendUint16(b, uint16(4+len(payload)))
	b = binary.LittleEndian.AppendUint16(b, typ)
	return append(b, payload...)
}

// U32 is a scalar record of type 0x0003.
func U32(key string, v uint32) []byte {
	return Record(key, 0x0003, binary.LittleEndian.AppendUint32(nil, v))
}

// Array is a counted u32 array record.
func Array(key string, typ uint16, vals ...uint32) []byte {
	p := binary.LittleEndian.AppendUint32(nil, uint32(len(vals)))
	for _, v := range vals {
		p = binary.LittleEndian.AppendUint32(p, v)
	}
	return Record(key, typ, p)
}

// Text is a UTF-16LE string record.
func Text(key, s string) []byte {
	u := utf16.Encode([]rune(s + "\x00"))
	p := binary.LittleEndian.AppendUint32(nil, uint32(len(u)))
	for _, c := range u {
		p = binary.LittleEndian.AppendUint16(p, c)
	}
	return Record(key, 0x001F, p)
}

// Filetime is a Windows FILETIME record.
func Filetime(key string, t time.Time) []byte {
	ticks := uint64(t.Unix()+11644473600)*10_000_000 + uint64(t.Nanosecond()/100)
	return Record(key, 0x0040, binary.LittleEndian.AppendUint64(nil, ticks))
}

// ScanDate is the date written by Inf.
var ScanDate = time.Date(2026, 11, 26, 10, 30, 0, 0, time.UTC)

// Inf is a typical set of frame records behind a few arbitrary bytes.
func Inf(w, h int, frame string, rotation uint32) []byte {
	b := []byte("XXXX\x00\x00\x00\x00")
	for _, r := range [][]byte{
		Text("00_MODEL_NAME", "SP-3000"),
		Array("02_FILE_SIZE", 0x1013, uint32(h), uint32(w)),
		U32("02_ROTATION_FOR_PROPER_RECTANGLE", rotation),
		Text("02_FRAME_NUMBER_STRING", frame),
		Filetime("02_SCANNING_DATE", ScanDate),
	} {
		b = append(b, r...)
	}
	return b
}
