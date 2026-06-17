package remote

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestValidateTarget(t *testing.T) {
	ok := []string{"root@host", "ubuntu@web-1", "deploy@pulse.tailnet.ts.net", "u_1@10.0.0.1"}
	for _, s := range ok {
		if !ValidateTarget(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	bad := []string{
		"root@host; rm -rf /", // injection
		"root@host && curl evil",
		"root@host`whoami`",
		"$(reboot)@host",
		"roothost", // no @
		"root@",
		"@host",
		"root@host space",
	}
	for _, s := range bad {
		if ValidateTarget(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestRunRejectsBadTarget(t *testing.T) {
	e := NewTailscaleSSH()
	if _, err := e.Run(context.Background(), "evil; rm -rf /", "echo hi"); err != ErrBadTarget {
		t.Fatalf("expected ErrBadTarget, got %v", err)
	}
}

func TestRunExecutesViaConfiguredCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell as the fake SSH command")
	}
	// Use `sh -c` as the "ssh" base: it receives <target> <command> as $0 $1.
	t.Setenv("REMOTE_SSH_CMD", "/bin/sh -c")
	e := NewTailscaleSSH()
	// command runs as: sh -c "echo reached" "root@host"  -> prints "reached"
	out, err := e.Run(context.Background(), "root@host", "echo reached")
	if err != nil {
		t.Fatalf("run error: %v (out=%q)", err, out)
	}
	if !strings.Contains(out, "reached") {
		t.Fatalf("expected output to contain 'reached', got %q", out)
	}
}
