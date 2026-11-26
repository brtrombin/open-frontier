package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/brtrombin/open-frontier/src/internal/fdia"
)

// cmdDB checks that the order database can be read from this PC, and shows
// the latest orders. Read-only.
func cmdDB(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("db", flag.ContinueOnError)
	db := addDBFlags(fs)
	n := fs.Int("orders", 15, "")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 && *db.host == "" {
		*db.host = pos[0]
	}
	cfg := db.config()
	if cfg == nil {
		return usageError("db needs the computer that holds the order database: openfrontier db EZPC00")
	}
	how := "TCP port " + fmt.Sprint(*db.port)
	if cfg.Pipe {
		how = `named pipe \\` + cfg.Host + `\pipe\sql\query`
	}
	fmt.Fprintf(out, "Connecting to %s on %s (%s), read-only...\n", cfg.Database, cfg.Host, how)
	conn, err := fdia.Open(*cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	if v, err := conn.Version(); err == nil {
		fmt.Fprintf(out, "server: %s\n", strings.Join(strings.Fields(v), " "))
	}
	orders, err := conn.Recent(*n)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\n%-10s %-18s %-7s %-22s %s\n", "folder", "order", "status", "meaning", "date and time")
	for _, o := range orders {
		fmt.Fprintf(out, "%08d   %-18s %-7d %-22s %s %s\n", o.ID, o.OrderID, o.Status, fdia.StatusName(o.Status), o.Date, o.Time)
	}
	fmt.Fprintf(out, "\n%d orders shown. Nothing was changed.\n", len(orders))
	return nil
}
