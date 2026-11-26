package imaging

import "encoding/binary"

var exifOrientations = map[uint16]string{
	1: "as-is", 2: "mirror", 3: "rot180", 4: "mirror+rot180",
	5: "mirror+rot90ccw", 6: "rot90cw", 7: "mirror+rot90cw", 8: "rot90ccw",
}

// ExifOrientation is the orientation that displays a JPEG as intended, from
// its EXIF Orientation tag, or "" when the file has none.
func ExifOrientation(jpeg []byte) string {
	if len(jpeg) < 4 || jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		return ""
	}
	pos := 2
	for pos+4 <= len(jpeg) {
		if jpeg[pos] != 0xFF {
			return ""
		}
		marker := jpeg[pos+1]
		if marker == 0xFF {
			pos++
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return ""
		}
		n := int(binary.BigEndian.Uint16(jpeg[pos+2:]))
		if n < 2 || pos+2+n > len(jpeg) {
			return ""
		}
		seg := jpeg[pos+4 : pos+2+n]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		pos += 2 + n
	}
	return ""
}

func tiffOrientation(t []byte) string {
	if len(t) < 8 {
		return ""
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return ""
	}
	ifd := int64(bo.Uint32(t[4:]))
	if ifd < 8 || ifd+2 > int64(len(t)) {
		return ""
	}
	count := int64(bo.Uint16(t[ifd:]))
	for i := int64(0); i < count; i++ {
		e := ifd + 2 + 12*i
		if e+12 > int64(len(t)) {
			return ""
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			return exifOrientations[bo.Uint16(t[e+8:])]
		}
	}
	return ""
}
