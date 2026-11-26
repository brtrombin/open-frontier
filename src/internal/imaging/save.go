package imaging

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
)

// Save writes img as "jpeg" or "tiff" through a temporary file, so that a
// failed write never leaves a partial image under the final name.
func Save(path string, img *image.RGBA, format string, quality int, deflate bool) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".openfrontier-*.tmp")
	if err != nil {
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	bw := bufio.NewWriterSize(tmp, 1<<20)
	switch format {
	case "jpeg":
		err = jpeg.Encode(bw, img, &jpeg.Options{Quality: quality})
	case "tiff":
		err = WriteTIFF(bw, img, deflate)
	default:
		err = fmt.Errorf("unknown format %q", format)
	}
	if err == nil {
		err = bw.Flush()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}
