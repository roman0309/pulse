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
		t.Skip("relies on a POSIX `echo` binary")
	}
	// Use `echo` as the fake SSH base: it runs `echo <target> <command>` and
	// echoes them back, so we can confirm the executor passes both through in
	// the right order regardless of any shell semantics.
	t.Setenv("REMOTE_SSH_CMD", "echo")
	e := NewTailscaleSSH()
	out, err := e.Run(context.Background(), "root@host", "install agent")
	if err != nil {
		t.Fatalf("run error: %v (out=%q)", err, out)
	}
	if !strings.Contains(out, "root@host") || !strings.Contains(out, "install agent") {
		t.Fatalf("expected output to contain target and command, got %q", out)
	}
}
