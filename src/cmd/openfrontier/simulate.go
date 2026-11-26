package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/brtrombin/open-frontier/src/internal/spool"
	"github.com/brtrombin/open-frontier/src/internal/synth"
)

var simulatedSizes = map[string]spool.Size{
	"small":  {W: 600, H: 400},
	"normal": {W: 3637, H: 2503},
	"large":  {W: 5444, H: 3649},
}

// cmdSimulate writes synthetic orders into a test folder the way a scanner
// fills its spool: each .RAW grows in pieces, its .INF comes after it. It
// lets watch be tried without a scanner, and refuses any real spool.
func cmdSimulate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("simulate", flag.ContinueOnError)
	frames := fs.Int("frames", 4, "")
	orders := fs.Int("orders", 1, "")
	size := fs.String("size", "small", "")
	seconds := fs.Float64("seconds", 2, "")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	dims, ok := simulatedSizes[*size]
	if len(pos) != 1 || !ok || *frames < 1 || *orders < 1 {
		return usageError("simulate needs a test folder; --size is small, normal or large")
	}
	dir := pos[0]
	if r := spool.LiveSpoolReason(dir); r != "" {
		return refuse("%s: %s; simulate only writes into test folders, never into a scanner spool", dir, r)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	pixels := synth.Raw(synth.Scene(dims.W, dims.H), dims.W, dims.H)
	pause := time.Duration(*seconds * float64(time.Second) / 8)
	for o := 0; o < *orders; o++ {
		id := 1
		for {
			if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("%08d", id))); os.IsNotExist(err) {
				break
			}
			id++
		}
		order := filepath.Join(dir, fmt.Sprintf("%08d", id))
		if err := os.MkdirAll(order, 0o755); err != nil {
			return err
		}
		fmt.Fprintf(out, "order %08d: writing %d frames of %s\n", id, *frames, dims)
		for f := 1; f <= *frames; f++ {
			raw := filepath.Join(order, fmt.Sprintf("%08d.RAW", f))
			if err := writeInPieces(raw, pixels, 8, pause); err != nil {
				return err
			}
			rotation := uint32(0)
			if f%2 == 0 {
				rotation = 90
			}
			name := fmt.Sprintf("%02dA", f)
			if err := os.WriteFile(filepath.Join(order, fmt.Sprintf("%06d.INF", f)), synth.Inf(dims.W, dims.H, name, rotation), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(out, "  frame %s written (rotation %s)\n", name, strconv.Itoa(int(rotation)))
		}
	}
	fmt.Fprintln(out, "done")
	return nil
}

func writeInPieces(path string, data []byte, pieces int, pause time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	step := (len(data) + pieces - 1) / pieces
	for i := 0; i < len(data); i += step {
		end := i + step
		if end > len(data) {
			end = len(data)
		}
		if _, err := f.Write(data[i:end]); err != nil {
			f.Close()
			return err
		}
		f.Sync()
		time.Sleep(pause)
	}
	return f.Close()
}
