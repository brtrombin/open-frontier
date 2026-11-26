// Command openfrontier reads the frame files that the scanner PC of a Fuji
// Frontier SP-3000 writes into its spool, converts them to JPEG or TIFF,
// compares them with the Fuji software's own output, and lists the FireWire
// hardware that Windows sees. It never talks to the scanner.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/brtrombin/open-frontier/src/internal/firewire"
	"github.com/brtrombin/open-frontier/src/internal/imaging"
	"github.com/brtrombin/open-frontier/src/internal/spool"
)

const version = "0.1.0"

const usage = `openfrontier %s: read-only tools for Fuji Frontier SP-3000 spool files

Usage:
  openfrontier inspect <order folder | file.RAW | file.INF> [--inf-dir DIR] [--inf FILE]
  openfrontier convert <order folder> --out DIR [options]
  openfrontier compare <file.RAW> <Fuji output of the same frame> [--inf FILE]
  openfrontier firewire
  openfrontier version

Dropping an order folder or a file on openfrontier.exe runs inspect on it.

convert options:
  --format jpeg|tiff                 output format (default jpeg)
  --quality N                        JPEG quality, 1 to 100 (default 95)
  --tiff-compression none|deflate    (default none)
  --rotate inf|none|90|180|270       inf follows the frame's .INF (default);
                                     numbers are degrees clockwise
  --channel-order rgb|bgr            (default rgb)
  --inf-dir DIR                      where the .INF files are when they are not
                                     beside the .RAW files (OutSpool)
  --overwrite                        replace output files that already exist

Safety: this program never writes to, renames in or deletes from the folders
it reads. It refuses to read a folder that may be the scanner's live spool (a
network path, or a folder named InSpool, OutSpool or ImgExchange) unless you
add --live-spool-ok. Only do that when no scan is running and the order is
finished; working on a copy is better.
`

const liveHelp = `Reading a spool folder that the scanner may still own can abort the Fuji
software and wedge the scanner. Copy the finished order to a local disk first,
with no scan running. If this already is such a copy, or the order is finished
and the scanner is idle, add --live-spool-ok.`

// infRotations is our provisional reading of 02_ROTATION_FOR_PROPER_RECTANGLE:
// degrees clockwise. compare is how we check it against real frames.
var infRotations = map[uint32]string{0: "as-is", 90: "rot90cw", 180: "rot180", 270: "rot90ccw"}

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	if ownConsole() {
		fmt.Print("\nPress Enter to close this window.")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(code)
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func refuse(format string, a ...interface{}) error {
	return &exitError{2, "refused: " + fmt.Sprintf(format, a...)}
}

func usageError(format string, a ...interface{}) error {
	return &exitError{2, fmt.Sprintf(format, a...) + "\nRun 'openfrontier help' for help."}
}

var errSomeFailed = &exitError{1, ""}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stdout, usage, version)
		return 2
	}
	var err error
	switch args[0] {
	case "inspect":
		err = cmdInspect(args[1:], stdout)
	case "convert":
		err = cmdConvert(args[1:], stdout, stderr)
	case "compare":
		err = cmdCompare(args[1:], stdout)
	case "firewire":
		err = cmdFirewire(stdout)
	case "version", "--version":
		fmt.Fprintln(stdout, "openfrontier", version)
	case "help", "--help", "-h", "/?":
		fmt.Fprintf(stdout, usage, version)
	default:
		if _, statErr := os.Stat(args[0]); statErr != nil {
			err = usageError("unknown command %q", args[0])
			break
		}
		err = cmdInspect(args, stdout)
	}
	if err == nil {
		return 0
	}
	var e *exitError
	if errors.As(err, &e) {
		if e.msg != "" {
			fmt.Fprintln(stderr, e.msg)
		}
		return e.code
	}
	fmt.Fprintln(stderr, "error:", err)
	return 1
}

// parseArgs lets flags come before or after the paths, which the flag
// package alone does not.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		if f := fs.Lookup(name); f != nil {
			if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
				continue
			}
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		}
	}
	if err := fs.Parse(flags); err != nil {
		return nil, usageError("%s: %v", fs.Name(), err)
	}
	return pos, nil
}

func checkSource(path string, liveOK bool, out io.Writer) error {
	if path == "" {
		return nil
	}
	reason := spool.LiveSpoolReason(path)
	if reason == "" {
		return nil
	}
	if !liveOK {
		return refuse("%s: %s.\n%s", path, reason, liveHelp)
	}
	fmt.Fprintf(out, "warning: reading %s although %s (--live-spool-ok)\n", path, reason)
	if runtime.GOOS != "windows" {
		fmt.Fprintln(out, "warning: on this system the SMB client, not this program, chooses how shared files are opened")
	}
	return nil
}

func checkOutput(out string, sources ...string) error {
	o, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	for _, s := range sources {
		if s == "" {
			continue
		}
		a, err := filepath.Abs(s)
		if err != nil {
			return err
		}
		if samePath(o, a) || within(o, a) {
			return refuse("the output folder %s is inside %s; choose a folder elsewhere", out, s)
		}
	}
	if r := spool.SpoolNameReason(out); r != "" {
		return refuse("the output folder %s: %s; never write into a scanner spool", out, r)
	}
	return nil
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func within(child, parent string) bool {
	prefix := strings.TrimSuffix(parent, string(filepath.Separator)) + string(filepath.Separator)
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(child), strings.ToLower(prefix))
	}
	return strings.HasPrefix(child, prefix)
}

func readInf(path string) (spool.Inf, error) {
	if path == "" {
		return nil, nil
	}
	return spool.ReadInf(path)
}

// findInf looks for the .INF of one .RAW file, as FindFrames would.
func findInf(raw, infDir string) string {
	pairs, err := spool.FindFrames(filepath.Dir(raw), infDir)
	if err != nil {
		return ""
	}
	for _, p := range pairs {
		if strings.EqualFold(filepath.Base(p.Raw), filepath.Base(raw)) {
			return p.Inf
		}
	}
	return ""
}

func cmdInspect(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	infDir := fs.String("inf-dir", "", "")
	infFile := fs.String("inf", "", "")
	liveOK := fs.Bool("live-spool-ok", false, "")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError("inspect needs one order folder, .RAW or .INF file")
	}
	path := pos[0]
	for _, p := range []string{path, *infDir, *infFile} {
		if err := checkSource(p, *liveOK, out); err != nil {
			return err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return inspectDir(out, path, *infDir)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".inf":
		inf, err := spool.ReadInf(path)
		if err != nil {
			return err
		}
		printInf(out, path, inf)
		return nil
	case ".raw":
		inf := *infFile
		if inf == "" {
			inf = findInf(path, *infDir)
		}
		return printRaw(out, path, inf)
	}
	return usageError("%s: expected an order folder, a .RAW or a .INF file", path)
}

func inspectDir(out io.Writer, path, infDir string) error {
	dirs, err := spool.OrderDirs(path)
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		return fmt.Errorf("%s: no .RAW files here or in its subfolders", path)
	}
	var first spool.FramePair
	for _, d := range dirs {
		pairs, err := spool.FindFrames(d, infDir)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "ORDER %s: %d frames\n", filepath.Base(d), len(pairs))
		fmt.Fprintf(out, "  %-14s %10s  %-12s %-11s %-6s %4s  %s\n", "raw", "bytes", "inf", "size", "frame", "rot", "status")
		for _, p := range pairs {
			if first.Raw == "" {
				first = p
			}
			fmt.Fprintln(out, "  "+frameRow(p))
		}
		if others, err := spool.OtherFiles(d); err == nil && len(others) > 0 {
			fmt.Fprintf(out, "  other files: %s\n", strings.Join(others, ", "))
		}
	}
	fmt.Fprintln(out)
	return printRaw(out, first.Raw, first.Inf)
}

func frameRow(p spool.FramePair) string {
	h, err := spool.ReadHeader(p.Raw)
	if err != nil {
		return fmt.Sprintf("%-14s error: %v", filepath.Base(p.Raw), err)
	}
	infName, size, frame, rot := "-", "-", "-", "-"
	var dims *spool.Size
	if p.Inf != "" {
		infName = filepath.Base(p.Inf)
		inf, err := spool.ReadInf(p.Inf)
		if err != nil {
			return fmt.Sprintf("%-14s error reading %s: %v", filepath.Base(p.Raw), infName, err)
		}
		if dims = inf.ImageSize(); dims != nil {
			size = dims.String()
		}
		if f := inf.FrameName(); f != "" {
			frame = f
		}
		if r, ok := inf.Rotation(); ok {
			rot = fmt.Sprint(r)
		}
	}
	return fmt.Sprintf("%-14s %10d  %-12s %-11s %-6s %4s  %s",
		filepath.Base(p.Raw), h.FileSize, infName, size, frame, rot, spool.Status(h, dims))
}

func printRaw(out io.Writer, rawPath, infPath string) error {
	h, err := spool.ReadHeader(rawPath)
	if err != nil {
		return err
	}
	inf, err := readInf(infPath)
	if err != nil {
		return err
	}
	size := inf.ImageSize()
	words := make([]string, len(h.Words))
	for i, w := range h.Words {
		words[i] = fmt.Sprint(w)
	}
	magic := "DR, as expected"
	if !h.MagicOK() {
		magic = fmt.Sprintf("%q, UNEXPECTED (we expected DR)", h.Magic())
	}
	fmt.Fprintf(out, "RAW %s\n", rawPath)
	fmt.Fprintf(out, "  size on disk : %d bytes\n", h.FileSize)
	fmt.Fprintf(out, "  magic        : %s\n", magic)
	fmt.Fprintf(out, "  header words : %s\n", strings.Join(words, " "))
	fmt.Fprintf(out, "  header dims  : %s (words 4 and 5)\n", h.Dims())
	switch {
	case infPath == "":
		fmt.Fprintln(out, "  INF          : not found")
	case size == nil:
		fmt.Fprintf(out, "  INF          : %s, without a usable 02_FILE_SIZE\n", infPath)
	default:
		fmt.Fprintf(out, "  INF dims     : %s (02_FILE_SIZE, stored as height, width)\n", size)
	}
	if dims := spool.Candidates(h, size); len(dims) == 0 {
		fmt.Fprintf(out, "  status       : %s\n", spool.SizeProblem(h, size))
	} else {
		fmt.Fprintf(out, "  status       : complete (%d bytes = 32 + width x height x 3)\n", h.FileSize)
		if size != nil && !containsSize(dims, *size) {
			fmt.Fprintln(out, "  warning      : the INF dimensions do not fit this file")
		}
		fr, err := spool.OpenFrame(rawPath, size)
		if err != nil {
			return err
		}
		for i, l := range fr.Layouts {
			label := "      "
			if i == 0 {
				label = "best  "
			}
			fmt.Fprintf(out, "  layout %s: %s, row difference %.1f\n", label, l, l.Score)
		}
		if fr.Ambiguous() {
			fmt.Fprintln(out, "  warning      : the layout is not clear-cut (flat or blank frame?); use compare to be sure")
		}
	}
	if inf != nil {
		fmt.Fprintln(out)
		printInf(out, infPath, inf)
	}
	return nil
}

func containsSize(list []spool.Size, s spool.Size) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func printInf(out io.Writer, path string, inf spool.Inf) {
	fields := inf.Sorted()
	fmt.Fprintf(out, "INF %s: %d fields\n", path, len(fields))
	for _, f := range fields {
		t := "type ?     "
		if f.Known {
			t = fmt.Sprintf("type 0x%04x", f.Type)
		}
		fmt.Fprintf(out, "  %6d  %s  size %5d  %-36s %s\n", f.Offset, t, f.RecordSize, f.Key, spool.FormatValue(f))
	}
}

type convertOptions struct {
	format, ext string
	quality     int
	deflate     bool
	rotate      string
	bgr         bool
	overwrite   bool
}

func cmdConvert(args []string, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	outDir := fs.String("out", "", "")
	infDir := fs.String("inf-dir", "", "")
	format := fs.String("format", "jpeg", "")
	quality := fs.Int("quality", 95, "")
	tiffComp := fs.String("tiff-compression", "none", "")
	rotate := fs.String("rotate", "inf", "")
	channels := fs.String("channel-order", "rgb", "")
	overwrite := fs.Bool("overwrite", false, "")
	liveOK := fs.Bool("live-spool-ok", false, "")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 || *outDir == "" {
		return usageError("convert needs one order folder and --out DIR")
	}
	opts := convertOptions{quality: *quality, rotate: strings.ToLower(*rotate), overwrite: *overwrite}
	switch strings.ToLower(*format) {
	case "jpeg", "jpg":
		opts.format, opts.ext = "jpeg", ".jpg"
	case "tiff", "tif":
		opts.format, opts.ext = "tiff", ".tif"
	default:
		return usageError("--format must be jpeg or tiff")
	}
	switch strings.ToLower(*tiffComp) {
	case "none":
	case "deflate":
		opts.deflate = true
	default:
		return usageError("--tiff-compression must be none or deflate")
	}
	switch opts.rotate {
	case "inf", "none", "0", "90", "180", "270":
	default:
		return usageError("--rotate must be inf, none, 90, 180 or 270")
	}
	switch strings.ToLower(*channels) {
	case "rgb":
	case "bgr":
		opts.bgr = true
	default:
		return usageError("--channel-order must be rgb or bgr")
	}
	if opts.quality < 1 || opts.quality > 100 {
		return usageError("--quality must be between 1 and 100")
	}
	src := pos[0]
	for _, p := range []string{src, *infDir} {
		if err := checkSource(p, *liveOK, out); err != nil {
			return err
		}
	}
	if err := checkOutput(*outDir, src, *infDir); err != nil {
		return err
	}
	dirs, err := spool.OrderDirs(src)
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		return fmt.Errorf("%s: no .RAW files here or in its subfolders", src)
	}
	written, kept, failed := 0, 0, 0
	for _, d := range dirs {
		order := filepath.Base(d)
		target := filepath.Join(*outDir, order)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		pairs, err := spool.FindFrames(d, *infDir)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "ORDER %s: %d frames -> %s\n", order, len(pairs), target)
		used := map[string]bool{}
		for _, p := range pairs {
			done, err := convertFrame(out, p, order, target, used, opts)
			switch {
			case err != nil:
				failed++
				fmt.Fprintf(errOut, "  skipped %s: %v\n", filepath.Base(p.Raw), err)
			case done:
				written++
			default:
				kept++
			}
		}
	}
	fmt.Fprintf(out, "done: %d written, %d already there, %d skipped\n", written, kept, failed)
	if failed > 0 {
		return errSomeFailed
	}
	return nil
}

func convertFrame(out io.Writer, p spool.FramePair, order, target string, used map[string]bool, o convertOptions) (bool, error) {
	inf, err := readInf(p.Inf)
	if err != nil {
		return false, err
	}
	stem := strings.TrimSuffix(filepath.Base(p.Raw), filepath.Ext(p.Raw))
	name := outputName(order, inf.FrameName(), stem, used)
	dest := filepath.Join(target, name+o.ext)
	if !o.overwrite {
		if _, err := os.Stat(dest); err == nil {
			fmt.Fprintf(out, "  %s: %s already exists, kept (use --overwrite)\n", filepath.Base(p.Raw), filepath.Base(dest))
			return false, nil
		}
	}
	fr, err := spool.OpenFrame(p.Raw, inf.ImageSize())
	if err != nil {
		return false, err
	}
	orientation, note := chooseOrientation(o.rotate, inf, p.Inf != "")
	img, err := imaging.Orient(imaging.FromFrame(fr, o.bgr), orientation)
	if err != nil {
		return false, err
	}
	if err := imaging.Save(dest, img, o.format, o.quality, o.deflate); err != nil {
		return false, err
	}
	notes := []string{fr.Best().String(), orientation}
	if note != "" {
		notes = append(notes, note)
	}
	if fr.Ambiguous() {
		notes = append(notes, "layout not clear-cut")
	}
	fmt.Fprintf(out, "  %s -> %s (%dx%d; %s)\n", filepath.Base(p.Raw), filepath.Base(dest),
		img.Rect.Dx(), img.Rect.Dy(), strings.Join(notes, "; "))
	return true, nil
}

func chooseOrientation(rotate string, inf spool.Inf, haveInf bool) (string, string) {
	switch rotate {
	case "none", "0":
		return "as-is", ""
	case "90":
		return "rot90cw", ""
	case "180":
		return "rot180", ""
	case "270":
		return "rot90ccw", ""
	}
	v, ok := inf.Rotation()
	switch {
	case !haveInf:
		return "as-is", "no .INF, not rotated"
	case !ok:
		return "as-is", "no rotation in the .INF, not rotated"
	}
	if name, known := infRotations[v]; known {
		return name, fmt.Sprintf("INF rotation %d", v)
	}
	return "as-is", fmt.Sprintf("unknown INF rotation value %d, not rotated", v)
}

func outputName(order, frame, stem string, used map[string]bool) string {
	base := sanitize(frame)
	if base == "" {
		base = stem
	}
	name := order + "-" + base
	if used[strings.ToLower(name)] {
		name += "-" + stem
	}
	used[strings.ToLower(name)] = true
	return name
}

func sanitize(s string) string {
	b := []byte(s)
	for i, c := range b {
		if !(c == '-' || c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			b[i] = '_'
		}
	}
	return strings.Trim(string(b), "_")
}

func cmdCompare(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	infFile := fs.String("inf", "", "")
	liveOK := fs.Bool("live-spool-ok", false, "")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return usageError("compare needs a .RAW file and the Fuji output of the same frame")
	}
	raw, refPath := pos[0], pos[1]
	for _, p := range []string{raw, *infFile} {
		if err := checkSource(p, *liveOK, out); err != nil {
			return err
		}
	}
	infPath := *infFile
	if infPath == "" {
		infPath = findInf(raw, "")
	}
	inf, err := readInf(infPath)
	if err != nil {
		return err
	}
	fr, err := spool.OpenFrame(raw, inf.ImageSize())
	if err != nil {
		return err
	}
	ref, exif, err := imaging.LoadReference(refPath)
	if err != nil {
		return err
	}
	matches := imaging.Rank(imaging.FromFrame(fr, false), ref)
	refNote := ""
	if exif != "" {
		refNote = ", EXIF orientation applied: " + exif
	}
	fmt.Fprintf(out, "RAW        %s: %s\n", filepath.Base(raw), fr.Best())
	fmt.Fprintf(out, "reference  %s: %dx%d%s\n", filepath.Base(refPath), ref.Rect.Dx(), ref.Rect.Dy(), refNote)
	fmt.Fprintln(out, "best matches (mean difference per channel, 0 to 255):")
	for _, m := range matches[:4] {
		order := "rgb"
		if m.BGR {
			order = "bgr"
		}
		fmt.Fprintf(out, "  %7.2f  %-16s %s\n", m.Difference, m.Orientation, order)
	}
	fmt.Fprintf(out, "verdict    %s\n", imaging.Verdict(matches))
	best := matches[0]
	w, h := fr.Best().W, fr.Best().H
	if strings.Contains(best.Orientation, "90") {
		w, h = h, w
	}
	fmt.Fprintf(out, "sizes      ours %dx%d after %s, reference %dx%d\n", w, h, best.Orientation, ref.Rect.Dx(), ref.Rect.Dy())
	if v, ok := inf.Rotation(); ok {
		fmt.Fprintf(out, "INF        02_ROTATION_FOR_PROPER_RECTANGLE = %d\n", v)
	} else {
		fmt.Fprintln(out, "INF        no rotation value available")
	}
	return nil
}

func cmdFirewire(out io.Writer) error {
	fmt.Fprintln(out, "Asking Windows for its FireWire (IEEE 1394) devices; this can take a few seconds.")
	fmt.Fprintln(out)
	inv, err := firewire.Collect()
	if err != nil {
		return err
	}
	firewire.Report(out, inv)
	return nil
}
