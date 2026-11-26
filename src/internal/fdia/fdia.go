// Package fdia reads the order database (FDIA_DB) of a Fuji Frontier system.
// It only ever runs the SELECT statements below, with NOLOCK, so it never
// writes and never holds a lock the scanner software could wait on.
// Schema notes: docs/protocol.md, section 9.5.
package fdia

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Config says how to reach the database.
type Config struct {
	Host     string
	Port     int // 0 means 1433
	Pipe     bool
	User     string
	Password string
	Database string
	Timeout  time.Duration
}

// DSN is the connection string for the go-mssqldb driver. With Pipe set it
// uses the named pipe \\host\pipe\sql\query (Windows only), which MSDE
// accepts by default; TCP is off by default on MSDE.
func (c Config) DSN() string {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	secs := strconv.Itoa(int(timeout / time.Second))
	q := url.Values{}
	q.Set("database", c.Database)
	q.Set("encrypt", "disable")
	q.Set("dial timeout", secs)
	q.Set("connection timeout", secs)
	q.Set("app name", "openfrontier (read-only)")
	u := &url.URL{Scheme: "sqlserver", User: url.UserPassword(c.User, c.Password)}
	if c.Pipe {
		u.Host = c.Host
		q.Set("protocol", "np")
		q.Set("pipe", `sql\query`)
	} else {
		port := c.Port
		if port == 0 {
			port = 1433
		}
		u.Host = net.JoinHostPort(c.Host, strconv.Itoa(port))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Order is one row of OrderTable.
type Order struct {
	ID      int
	OrderID string
	Status  int
	Date    string
	Time    string
}

// StatusName gives the meaning of the OrderTable status codes we know.
func StatusName(s int) string {
	switch s {
	case 0:
		return "error"
	case 2:
		return "scan finished"
	case 4:
		return "waiting to write"
	case 5:
		return "completed"
	}
	return "unknown"
}

// DB is an open, read-only connection.
type DB struct {
	db      *sql.DB
	timeout time.Duration
}

// Open connects and checks the connection.
func Open(c Config) (*DB, error) {
	db, err := sql.Open("sqlserver", c.DSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	d := &DB{db: db, timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot reach the order database on %s: %v", c.Host, err)
	}
	return d, nil
}

// Close ends the connection.
func (d *DB) Close() error { return d.db.Close() }

// Version is the server's @@VERSION text.
func (d *DB) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	var v string
	err := d.db.QueryRowContext(ctx, "SELECT @@VERSION").Scan(&v)
	return v, err
}

// Orders reads every order's id, name and status.
func (d *DB) Orders() (map[int]Order, error) {
	list, err := d.query("SELECT FDIAManageID, OrderID, Status, [Date], [Time] FROM OrderTable WITH (NOLOCK)")
	if err != nil {
		return nil, err
	}
	out := make(map[int]Order, len(list))
	for _, o := range list {
		out[o.ID] = o
	}
	return out, nil
}

// Recent reads the n orders with the highest ids.
func (d *DB) Recent(n int) ([]Order, error) {
	return d.query(fmt.Sprintf("SELECT TOP %d FDIAManageID, OrderID, Status, [Date], [Time] FROM OrderTable WITH (NOLOCK) ORDER BY FDIAManageID DESC", n))
}

func (d *DB) query(q string) ([]Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		var (
			id         int
			name, date sql.NullString
			hour       sql.NullString
			status     sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &status, &date, &hour); err != nil {
			return nil, err
		}
		s := -1
		if status.Valid {
			s = int(status.Int64)
		}
		out = append(out, Order{ID: id, OrderID: name.String, Status: s, Date: date.String, Time: hour.String})
	}
	return out, rows.Err()
}
