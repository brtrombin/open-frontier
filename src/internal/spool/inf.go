package spool

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Type codes seen in DIPP records.
const (
	TypeU32        = 0x0003
	TypeU8         = 0x0008
	TypeU32OrArray = 0x0013
	TypeString     = 0x001F
	TypeFiletime   = 0x0040
	TypeArray      = 0x1013
)

var knownTypes = map[uint16]bool{
	TypeU32: true, TypeU8: true, TypeU32OrArray: true,
	TypeString: true, TypeFiletime: true, TypeArray: true,
}

// arrayKeys resolves type 0x0013, which holds a scalar for most keys.
var arrayKeys = map[string]bool{"02_SOURCE_SIZE": true, "02_SCAN_SIZE": true, "02_FILE_SIZE": true}

// Field is one key of a .INF file and the record that follows it.
type Field struct {
	Key        string
	Offset     int // start of the record header, after the key
	RecordSize uint16
	Type       uint16
	Known      bool        // a record header with a known type code was found
	Value      interface{} // uint32, uint8, string, []uint32, time.Time or nil
	Raw        []byte      // first bytes after the record header, for reports
}

// Inf is a parsed .INF file, by key.
type Inf map[string]Field

// ReadInf reads and parses a .INF file.
func ReadInf(path string) (Inf, error) {
	data, err := ReadFileShared(path)
	if err != nil {
		return nil, err
	}
	return ParseInf(data), nil
}

// ParseInf finds every "NN_NAME" key ending in a NUL byte and decodes the
// record after it. Unknown keys are kept, so reports show all of them.
func ParseInf(data []byte) Inf {
	inf := Inf{}
	for i, b := range data {
		if b != 0 {
			continue
		}
		key, ok := keyEndingAt(data, i)
		if !ok {
			continue
		}
		if _, seen := inf[key]; !seen {
			inf[key] = decodeField(data, key, i+1)
		}
	}
	return inf
}

func keyEndingAt(data []byte, nul int) (string, bool) {
	start := nul
	for start > 0 && nul-start < 80 && isKeyChar(data[start-1]) {
		start--
	}
	for k := start; k+5 <= nul; k++ {
		if isDigit(data[k]) && isDigit(data[k+1]) && data[k+2] == '_' && nul-k <= 66 {
			return string(data[k:nul]), true
		}
	}
	return "", false
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isKeyChar(b byte) bool {
	return isDigit(b) || b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func decodeField(data []byte, key string, pos int) Field {
	f := Field{Key: key, Offset: pos}
	start, ok := recordStart(data, pos)
	if !ok {
		f.Raw = clip(data, pos, 16)
		return f
	}
	f.Offset = start
	f.RecordSize = binary.LittleEndian.Uint16(data[start:])
	f.Type = binary.LittleEndian.Uint16(data[start+2:])
	f.Known = true
	f.Raw = clip(data, start+4, 16)
	f.Value = decodeValue(data, start, key, f.Type)
	return f
}

// recordStart finds the record header after a key, skipping NUL padding.
func recordStart(data []byte, pos int) (int, bool) {
	for s := pos; s < pos+4 && s+4 <= len(data); s++ {
		if knownTypes[binary.LittleEndian.Uint16(data[s+2:])] {
			return s, true
		}
		if data[s] != 0 {
			break
		}
	}
	return 0, false
}

func clip(data []byte, from, n int) []byte {
	if from >= len(data) {
		return nil
	}
	end := from + n
	if end > len(data) {
		end = len(data)
	}
	return data[from:end]
}

func decodeValue(data []byte, s int, key string, t uint16) interface{} {
	u32 := func(at int) (uint32, bool) {
		if at+4 > len(data) {
			return 0, false
		}
		return binary.LittleEndian.Uint32(data[at:]), true
	}
	switch {
	case t == TypeString:
		n, ok := u32(s + 4)
		if !ok {
			return nil
		}
		if n > 4096 {
			n = 4096
		}
		var units []uint16
		for p := s + 8; p+2 <= len(data) && p < s+8+2*int(n); p += 2 {
			u := binary.LittleEndian.Uint16(data[p:])
			if u == 0 {
				break
			}
			units = append(units, u)
		}
		return string(utf16.Decode(units))
	case t == TypeFiletime:
		if s+12 > len(data) {
			return nil
		}
		ticks := binary.LittleEndian.Uint64(data[s+4:])
		if ticks == 0 {
			return nil
		}
		return fromFiletime(ticks)
	case t == TypeArray || (t == TypeU32OrArray && arrayKeys[key]):
		n, ok := u32(s + 4)
		if !ok || n > 64 || s+8+4*int(n) > len(data) {
			return nil
		}
		vals := make([]uint32, n)
		for i := range vals {
			vals[i] = binary.LittleEndian.Uint32(data[s+8+4*i:])
		}
		return vals
	case t == TypeU32 || t == TypeU32OrArray:
		if v, ok := u32(s + 4); ok {
			return v
		}
	case t == TypeU8:
		if s+4 < len(data) {
			return data[s+4]
		}
	}
	return nil
}

func fromFiletime(ticks uint64) time.Time {
	const unixOffset = 11644473600 // seconds from 1601-01-01 to 1970-01-01
	return time.Unix(int64(ticks/10_000_000)-unixOffset, int64(ticks%10_000_000)*100).UTC()
}

// ImageSize is (width, height) from 02_FILE_SIZE, which stores (height, width).
func (inf Inf) ImageSize() *Size {
	v, ok := inf["02_FILE_SIZE"].Value.([]uint32)
	if !ok || len(v) < 2 || v[0] == 0 || v[1] == 0 || v[0] > 1<<16 || v[1] > 1<<16 {
		return nil
	}
	return &Size{int(v[1]), int(v[0])}
}

// Rotation is the value of 02_ROTATION_FOR_PROPER_RECTANGLE.
func (inf Inf) Rotation() (uint32, bool) {
	v, ok := inf["02_ROTATION_FOR_PROPER_RECTANGLE"].Value.(uint32)
	return v, ok
}

// FrameName is the film frame designation, such as "01A".
func (inf Inf) FrameName() string {
	v, _ := inf["02_FRAME_NUMBER_STRING"].Value.(string)
	return strings.TrimSpace(v)
}

// Sorted lists the fields in file order.
func (inf Inf) Sorted() []Field {
	out := make([]Field, 0, len(inf))
	for _, f := range inf {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Offset < out[j].Offset })
	return out
}

// FormatValue renders a field value for reports.
func FormatValue(f Field) string {
	switch v := f.Value.(type) {
	case nil:
		return fmt.Sprintf("? % x", f.Raw)
	case string:
		return strconv.Quote(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05") + " (FILETIME)"
	case []uint32:
		parts := make([]string, len(v))
		for i, x := range v {
			parts[i] = strconv.FormatUint(uint64(x), 10)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case uint32:
		return fmt.Sprintf("%d (0x%x)", v, v)
	case uint8:
		return fmt.Sprintf("%d (u8)", v)
	}
	return fmt.Sprint(f.Value)
}
