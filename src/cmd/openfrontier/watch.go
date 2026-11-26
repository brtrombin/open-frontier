package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/brtrombin/open-frontier/src/internal/fdia"
	"github.com/brtrombin/open-frontier/src/internal/spool"
)

// dbFlags are the order-database options shared by watch and db.
type dbFlags struct {
	host, user, password, database *string
	port                           *int
	pipe                           *bool
}

func addDBFlags(fs *flag.FlagSet) dbFlags {
	return dbFlags{
		host:     fs.String("db-host", "", ""),
		user:     fs.String("db-user", "fdiaclient", ""),
		password: fs.String("db-password", "radiogaga", ""),
		database: fs.String("db-name", "FDIA_DB", ""),
		port:     fs.Int("db-port", 1433, ""),
		pipe:     fs.Bool("db-pipe", false, ""),
	}
}

func (f dbFlags) config() *fdia.Config {
	if *f.host == "" {
		return nil
	}
	return &fdia.Config{Host: *f.host, Port: *f.port, Pipe: *f.pipe, User: *f.user, Password: *f.password, Database: *f.database}
}

// orderGate decides when an order folder may be read, and its delivered name.
type orderGate interface {
	refresh() error
	check(folder string, newest time.Time) (ready bool, name, why string)
}

// dbGate follows OrderTable.Status: the documented safe signal.
type dbGate struct {
	db interface {
		Orders() (map[int]fdia.Order, error)
	}
	ready  int
	orders map[int]fdia.Order
}

func (g *dbGate) refresh() error {
	o, err := g.db.Orders()
	if err == nil {
		g.orders = o
	}
	return err
}

func (g *dbGate) check(folder string, _ time.Time) (bool, string, string) {
	id, err := strconv.Atoi(folder)
	if err != nil {
		return false, "", "folder name is not an order number"
	}
	o, ok := g.orders[id]
	if !ok {
		return false, "", "not in the order database"
	}
	if o.Status != g.ready {
		return false, "", fmt.Sprintf("status %d (%s), waiting", o.Status, fdia.StatusName(o.Status))
	}
	name := sanitize(o.OrderID)
	if name == "" {
		name = folder
	}
	return true, name, ""
}

// quietGate waits until nothing in the folder has changed for a while. It is
// only allowed on folders that are not a live spool, such as copies.
type quietGate struct {
	quiet time.Duration
	now   func() time.Time
}

func (quietGate) refresh() error { return nil }

func (g quietGate) check(folder string, newest time.Time) (bool, string, string) {
	if g.now().Sub(newest) < g.quiet {
		return false, "", fmt.Sprintf("still changing, waiting for %s without changes", g.quiet)
	}
	return true, folder, ""
}

type orderRecord struct {
	Name     string    `json:"name"`
	Status   string    `json:"status"` // done or failed
	Frames   int       `json:"frames"`
	Attempts int       `json:"attempts"`
	Note     string    `json:"note,omitempty"`
	When     time.Time `json:"when"`
}

type watcher struct {
	spool, infDir, out string
	conv               convertOptions
	gate               orderGate
	log                io.Writer
	state              map[string]*orderRecord
	pending            map[string]string
	beat               time.Time
}

const maxAttempts = 3

func (w *watcher) stateDir() string  { return filepath.Join(w.out, ".openfrontier") }
func (w *watcher) statePath() string { return filepath.Join(w.stateDir(), "state.json") }

func (w *watcher) logf(format string, a ...interface{}) {
	fmt.Fprintf(w.log, time.Now().Format("2006-01-02 15:04:05")+"  "+format+"\n", a...)
}

func (w *watcher) loadState() error {
	w.state = map[string]*orderRecord{}
	w.pending = map[string]string{}
	data, err := os.ReadFile(w.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &w.state)
}

func (w *watcher) saveState() error {
	data, err := json.MarshalIndent(w.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := w.statePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, w.statePath())
}

// orderKey tells a recycled order number apart: FDIAManageID restarts after
// 10000 orders, so the folder name alone is not unique over time.
func orderKey(dir string, files []os.FileInfo) string {
	var first time.Time
	for _, f := range files {
		if strings.EqualFold(filepath.Ext(f.Name()), ".raw") && (first.IsZero() || f.ModTime().Before(first)) {
			first = f.ModTime()
		}
	}
	return fmt.Sprintf("%s@%d", filepath.Base(dir), first.Unix())
}

// folderFiles stats the files of an order folder without opening any of them.
func folderFiles(dirs ...string) ([]os.FileInfo, time.Time) {
	var files []os.FileInfo
	var newest time.Time
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || info.IsDir() {
				continue
			}
			files = append(files, info)
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
		}
	}
	return files, newest
}

// note logs a message once, and again only when it changes; "" clears it.
func (w *watcher) note(key, msg string) {
	if msg == "" {
		if old, ok := w.pending[key]; ok && strings.HasPrefix(old, "!") {
			w.logf("%s is back", key)
		}
		delete(w.pending, key)
		return
	}
	if w.pending[key] != msg {
		w.logf("%s", strings.TrimPrefix(msg, "!"))
		w.pending[key] = msg
	}
}

func (w *watcher) pass() {
	if err := w.gate.refresh(); err != nil {
		w.note("order database", "!order database not reachable, nothing is read until it answers: "+err.Error())
		return
	}
	w.note("order database", "")
	dirs, err := spool.OrderDirs(w.spool)
	if err != nil {
		w.note("spool folder", "!spool folder not reachable: "+err.Error())
		return
	}
	w.note("spool folder", "")
	waiting := 0
	defer func() {
		if time.Since(w.beat) >= time.Minute {
			w.logf("heartbeat: %d order folders in the spool, %d waiting", len(dirs), waiting)
			w.beat = time.Now()
		}
	}()
	for _, dir := range dirs {
		folder := filepath.Base(dir)
		watched := []string{dir}
		if w.infDir != "" {
			watched = append(watched, filepath.Join(w.infDir, folder))
		}
		files, newest := folderFiles(watched...)
		key := orderKey(dir, files)
		if rec := w.state[key]; rec != nil && (rec.Status == "done" || rec.Status == "failed") {
			continue
		}
		ready, name, why := w.gate.check(folder, newest)
		if !ready {
			waiting++
			w.note(key, "order "+folder+": "+why)
			continue
		}
		delete(w.pending, key)
		rec := w.state[key]
		if rec == nil {
			rec = &orderRecord{}
			w.state[key] = rec
		}
		rec.Attempts++
		frames, final, err := w.deliver(dir, folder, name)
		rec.When = time.Now()
		if err != nil {
			rec.Note = err.Error()
			if rec.Attempts >= maxAttempts {
				rec.Status = "failed"
				w.logf("order %s: FAILED after %d attempts: %v", folder, rec.Attempts, err)
			} else {
				w.logf("order %s: %v (will retry)", folder, err)
			}
		} else {
			rec.Status, rec.Name, rec.Frames, rec.Note = "done", filepath.Base(final), frames, ""
			w.logf("order %s: delivered %d frames to %s", folder, frames, final)
		}
		if err := w.saveState(); err != nil {
			w.logf("cannot save state: %v", err)
		}
	}
}

// deliver copies an order to local staging, converts it, and moves the
// result into place in one rename, so readers of the output never see a
// partial order. Nothing in the spool is changed.
func (w *watcher) deliver(dir, folder, name string) (int, string, error) {
	pairs, err := spool.FindFrames(dir, w.infDir)
	if err != nil {
		return 0, "", err
	}
	if len(pairs) == 0 {
		return 0, "", errors.New("no frames")
	}
	stage := filepath.Join(w.stateDir(), "staging", folder)
	if err := os.RemoveAll(stage); err != nil {
		return 0, "", err
	}
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return 0, "", err
	}
	defer os.RemoveAll(stage)
	for _, p := range pairs {
		for _, src := range []string{p.Raw, p.Inf} {
			if src == "" {
				continue
			}
			if err := copyShared(src, filepath.Join(stage, filepath.Base(src))); err != nil {
				return 0, "", err
			}
		}
	}
	staged, err := spool.FindFrames(stage, "")
	if err != nil {
		return 0, "", err
	}
	work := filepath.Join(w.out, "_inprogress", name)
	if err := os.RemoveAll(work); err != nil {
		return 0, "", err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return 0, "", err
	}
	used := map[string]bool{}
	var failed []string
	for _, p := range staged {
		if _, err := convertFrame(io.Discard, p, name, work, used, w.conv); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", filepath.Base(p.Raw), err))
		}
	}
	if len(failed) > 0 {
		return 0, "", errors.New(strings.Join(failed, "; "))
	}
	final := filepath.Join(w.out, name)
	for n := 2; ; n++ {
		if _, err := os.Stat(final); errors.Is(err, os.ErrNotExist) {
			break
		}
		final = filepath.Join(w.out, fmt.Sprintf("%s-%d", name, n))
	}
	if err := os.Rename(work, final); err != nil {
		return 0, "", err
	}
	os.Remove(filepath.Join(w.out, "_inprogress"))
	return len(staged), final, nil
}

func copyShared(src, dst string) error {
	in, err := spool.OpenShared(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n != info.Size() {
		err = fmt.Errorf("%s changed size while it was copied", src)
	}
	return err
}

func cmdWatch(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	outDir := fs.String("out", "", "")
	infDir := fs.String("inf-dir", "", "")
	format := fs.String("format", "jpeg", "")
	quality := fs.Int("quality", 95, "")
	tiffComp := fs.String("tiff-compression", "none", "")
	rotate := fs.String("rotate", "inf", "")
	channels := fs.String("channel-order", "rgb", "")
	quiet := fs.Int("quiet", 150, "")
	poll := fs.Int("poll", 5, "")
	ready := fs.Int("ready-status", 2, "")
	once := fs.Bool("once", false, "")
	db := addDBFlags(fs)
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 || *outDir == "" {
		return usageError("watch needs the spool folder and --out DIR")
	}
	conv, err := makeConvertOptions(*format, *quality, *tiffComp, *rotate, *channels, false)
	if err != nil {
		return err
	}
	src := pos[0]
	if err := checkOutput(*outDir, src, *infDir); err != nil {
		return err
	}
	var gate orderGate
	cfg := db.config()
	if cfg == nil {
		for _, p := range []string{src, *infDir} {
			if r := spool.LiveSpoolReason(p); p != "" && r != "" {
				return refuse("%s: %s.\nWatching a live spool needs the order database (--db-host), so that an order is only read after the scanner marks it finished. Without it, watch only works on copies or test folders.", p, r)
			}
		}
		gate = quietGate{quiet: time.Duration(*quiet) * time.Second, now: time.Now}
	} else {
		for _, p := range []string{src, *infDir} {
			if err := checkSource(p, true, out); err != nil {
				return err
			}
		}
		conn, err := fdia.Open(*cfg)
		if err != nil {
			return err
		}
		defer conn.Close()
		gate = &dbGate{db: conn, ready: *ready}
	}
	if err := os.MkdirAll(filepath.Join(*outDir, ".openfrontier"), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(*outDir, ".openfrontier", "watch.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	w := &watcher{spool: src, infDir: *infDir, out: *outDir, conv: conv, gate: gate, log: io.MultiWriter(out, logFile)}
	if err := w.loadState(); err != nil {
		return fmt.Errorf("state file: %v", err)
	}
	mode := fmt.Sprintf("orders are read after %d s without changes (copies and test folders only)", *quiet)
	if cfg != nil {
		mode = fmt.Sprintf("orders are read when OrderTable.Status = %d on %s", *ready, cfg.Host)
	}
	w.logf("watching %s, delivering to %s; %s. Nothing in the spool is changed. Ctrl+C stops.", src, *outDir, mode)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)
	for {
		w.pass()
		if *once {
			return nil
		}
		select {
		case <-stop:
			w.logf("stopped")
			return nil
		case <-time.After(time.Duration(*poll) * time.Second):
		}
	}
}
