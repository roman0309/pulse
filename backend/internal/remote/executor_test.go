package remote

import (
	"context"
	"testing"
	"time"
)

func TestRunRejectsIncompleteConn(t *testing.T) {
	e := NewSSH()
	cases := []Conn{
		{},                                  // nothing
		{Host: "h"},                         // no user
		{Host: "h", User: "u"},              // no credentials
	}
	for _, c := range cases {
		if _, _, err := e.Run(context.Background(), c, "echo hi"); err != ErrBadConn {
			t.Errorf("conn %+v: expected ErrBadConn, got %v", c, err)
		}
	}
}

func TestRunUnreachableHostErrors(t *testing.T) {
	e := &SSHExecutor{DialTimeout: 500 * time.Millisecond}
	// RFC 5737 TEST-NET-1, not routable -> dial fails (not ErrBadConn).
	_, _, err := e.Run(context.Background(), Conn{Host: "192.0.2.1", User: "u", Password: "p"}, "echo hi")
	if err == nil || err == ErrBadConn {
		t.Fatalf("expected a dial error, got %v", err)
	}
}
