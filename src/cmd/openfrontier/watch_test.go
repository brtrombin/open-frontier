package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brtrombin/open-frontier/src/internal/fdia"
	"github.com/brtrombin/open-frontier/src/internal/synth"
)

func age(t *testing.T, dir string, d time.Duration) {
	t.Helper()
	old := time.Now().Add(-d)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if err := os.Chtimes(filepath.Join(dir, e.Name()), old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func snapshot(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			data, _ := os.ReadFile(p)
			out[p] = data
		}
		return nil
	})
	return out
}

func TestWatchQuietCopy(t *testing.T) {
	root := t.TempDir()
	spoolDir := filepath.Join(root, "copies")
	dir := order(t, spoolDir)
	age(t, dir, time.Hour)
	before := snapshot(t, spoolDir)
	out := filepath.Join(root, "ssd")

	code, stdout, stderr := runCmd("watch", spoolDir, "--out", out, "--once")
	if code != 0 || !strings.Contains(stdout, "delivered 2 frames") {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	if s := decodeSize(t, filepath.Join(out, "00006360", "00006360-01A.jpg")); s.X != h || s.Y != w {
		t.Fatalf("01A size %v", s)
	}
	if after := snapshot(t, spoolDir); len(after) != len(before) {
		t.Fatal("the spool changed")
	} else {
		for p, data := range before {
			if !bytes.Equal(after[p], data) {
				t.Fatalf("%s changed", p)
			}
		}
	}
	for _, leftover := range []string{"_inprogress/00006360", ".openfrontier/staging/00006360"} {
		if _, err := os.Stat(filepath.Join(out, leftover)); !os.IsNotExist(err) {
			t.Fatalf("%s left behind", leftover)
		}
	}
	if code, stdout, _ := runCmd("watch", spoolDir, "--out", out, "--once"); code != 0 || strings.Contains(stdout, "delivered") {
		t.Fatalf("delivered twice:\n%s", stdout)
	}

	// The same folder number reused later (FDIAManageID recycles) is a new order.
	os.RemoveAll(dir)
	dir = order(t, spoolDir)
	age(t, dir, 30*time.Minute)
	if code, stdout, _ := runCmd("watch", spoolDir, "--out", out, "--once"); code != 0 || !strings.Contains(stdout, "00006360-2") {
		t.Fatalf("recycled number:\n%s", stdout)
	}
}

func TestWatchWaitsWhileWriting(t *testing.T) {
	root := t.TempDir()
	spoolDir := filepath.Join(root, "test-spool")
	order(t, spoolDir)
	out := filepath.Join(root, "ssd")
	code, stdout, _ := runCmd("watch", spoolDir, "--out", out, "--once", "--quiet", "60")
	if code != 0 || !strings.Contains(stdout, "waiting for 1m0s without changes") || strings.Contains(stdout, "delivered") {
		t.Fatalf("%d\n%s", code, stdout)
	}
}

func TestWatchRefusesLiveSpoolWithoutDatabase(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "InSpool")
	order(t, live)
	code, _, stderr := runCmd("watch", live, "--out", filepath.Join(root, "ssd"), "--once")
	if code != 2 || !strings.Contains(stderr, "--db-host") {
		t.Fatalf("%d %s", code, stderr)
	}
	if code, _, _ := runCmd("watch", filepath.Join(root, "copies"), "--out", filepath.Join(root, "copies", "out")); code != 2 {
		t.Fatal("output inside the spool not refused")
	}
}

type fakeDB map[int]fdia.Order

func (f fakeDB) Orders() (map[int]fdia.Order, error) { return f, nil }

func TestWatchFollowsDatabaseStatus(t *testing.T) {
	root := t.TempDir()
	spoolDir := filepath.Join(root, "spool")
	order(t, spoolDir)
	out := filepath.Join(root, "ssd")
	os.MkdirAll(filepath.Join(out, ".openfrontier"), 0o755)
	db := fakeDB{6360: {ID: 6360, OrderID: "GP0814 G2", Status: 1}}
	var log bytes.Buffer
	conv, _ := makeConvertOptions("jpeg", 90, "none", "inf", "rgb", false)
	wt := &watcher{spool: spoolDir, out: out, conv: conv, gate: &dbGate{db: db, ready: 2}, log: &log}
	if err := wt.loadState(); err != nil {
		t.Fatal(err)
	}
	wt.pass()
	if !strings.Contains(log.String(), "status 1") || strings.Contains(log.String(), "delivered") {
		t.Fatalf("read before the scanner finished:\n%s", log.String())
	}
	db[6360] = fdia.Order{ID: 6360, OrderID: "GP0814 G2", Status: 2}
	wt.pass()
	if _, err := os.Stat(filepath.Join(out, "GP0814_G2", "GP0814_G2-02A.jpg")); err != nil {
		t.Fatalf("not delivered under the order name:\n%s", log.String())
	}
}

func TestWatchRetriesThenGivesUp(t *testing.T) {
	root := t.TempDir()
	spoolDir := filepath.Join(root, "copies")
	dir := order(t, spoolDir)
	data, _ := os.ReadFile(filepath.Join(dir, "00000002.RAW"))
	write(t, filepath.Join(dir, "00000002.RAW"), data[:len(data)-5])
	age(t, dir, time.Hour)
	out := filepath.Join(root, "ssd")
	var last string
	for i := 0; i < 4; i++ {
		_, last, _ = runCmd("watch", spoolDir, "--out", out, "--once")
	}
	if !strings.Contains(last, "") {
		t.Fatal()
	}
	state, _ := os.ReadFile(filepath.Join(out, ".openfrontier", "state.json"))
	if !strings.Contains(string(state), `"failed"`) || !strings.Contains(string(state), "short") {
		t.Fatalf("state:\n%s", state)
	}
	if _, err := os.Stat(filepath.Join(out, "00006360")); !os.IsNotExist(err) {
		t.Fatal("a partial order was delivered")
	}
}

func TestWatchInfInOutSpool(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "in", "00006360")
	write(t, filepath.Join(dir, "00000001.RAW"), synth.Raw(synth.Scene(w, h), w, h))
	outSpool := filepath.Join(root, "out-spool")
	write(t, filepath.Join(outSpool, "00006360", "000001.INF"), synth.Inf(w, h, "07A", 90))
	age(t, dir, time.Hour)
	age(t, filepath.Join(outSpool, "00006360"), time.Hour)
	out := filepath.Join(root, "ssd")
	code, stdout, stderr := runCmd("watch", filepath.Join(root, "in"), "--inf-dir", outSpool, "--out", out, "--once")
	if code != 0 {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	if s := decodeSize(t, filepath.Join(out, "00006360", "00006360-07A.jpg")); s.X != h {
		t.Fatalf("size %v", s)
	}
}

func TestSimulateThenWatch(t *testing.T) {
	root := t.TempDir()
	if code, _, _ := runCmd("simulate", filepath.Join(root, "InSpool")); code != 2 {
		t.Fatal("simulate wrote into a spool-named folder")
	}
	test := filepath.Join(root, "test-spool")
	code, stdout, stderr := runCmd("simulate", test, "--frames", "2", "--seconds", "0")
	if code != 0 {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	out := filepath.Join(root, "ssd")
	code, stdout, stderr = runCmd("watch", test, "--out", out, "--once", "--quiet", "0")
	if code != 0 || !strings.Contains(stdout, "delivered 2 frames") {
		t.Fatalf("%d\n%s\n%s", code, stdout, stderr)
	}
	if s := decodeSize(t, filepath.Join(out, "00000001", "00000001-02A.jpg")); s.X != 400 || s.Y != 600 {
		t.Fatalf("02A should be rotated: %v", s)
	}
}

func TestDBNeedsHost(t *testing.T) {
	if code, _, _ := runCmd("db"); code != 2 {
		t.Fatal("db without host")
	}
}
