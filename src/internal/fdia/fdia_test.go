package fdia

import (
	"net/url"
	"testing"
)

func TestDSN(t *testing.T) {
	tcp, err := url.Parse(Config{Host: "EZPC00", User: "u", Password: "p w", Database: "FDIA_DB"}.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if tcp.Host != "EZPC00:1433" || tcp.Query().Get("database") != "FDIA_DB" || tcp.Query().Get("encrypt") != "disable" {
		t.Fatalf("tcp DSN %v", tcp)
	}
	if pw, _ := tcp.User.Password(); pw != "p w" {
		t.Fatalf("password %q", pw)
	}
	np, _ := url.Parse(Config{Host: "EZPC00", Pipe: true, Database: "FDIA_DB"}.DSN())
	if np.Host != "EZPC00" || np.Query().Get("protocol") != "np" || np.Query().Get("pipe") != `sql\query` {
		t.Fatalf("pipe DSN %v", np)
	}
	if StatusName(2) != "scan finished" || StatusName(9) != "unknown" {
		t.Fatal("status names")
	}
}
