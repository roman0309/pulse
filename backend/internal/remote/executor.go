// Package remote runs commands on managed servers over Tailscale SSH.
// No credentials are stored — auth is the tailnet identity of the Pulse node.
package remote

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Executor runs a shell command on a remote host identified by an SSH target.
type Executor interface {
	Run(ctx context.Context, target, command string) (output string, err error)
}

// validTarget allows only "user@host" with safe characters (no shell metachars),
// preventing command injection through the target field.
var validTarget = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9._-]+$`)

// ValidateTarget reports whether target is a safe user@host value.
func ValidateTarget(target string) bool { return validTarget.MatchString(target) }

var ErrBadTarget = errors.New("invalid ssh target (expected user@host)")

// TailscaleSSH executes commands via the `tailscale ssh` CLI. The base command
// is configurable (REMOTE_SSH_CMD, default "tailscale ssh") so it can be pointed
// at a wrapper or a plain ssh client in testing.
type TailscaleSSH struct {
	base []string
}

func NewTailscaleSSH() *TailscaleSSH {
	base := os.Getenv("REMOTE_SSH_CMD")
	if base == "" {
		base = "tailscale ssh"
	}
	return &TailscaleSSH{base: strings.Fields(base)}
}

func (t *TailscaleSSH) Run(ctx context.Context, target, command string) (string, error) {
	if !ValidateTarget(target) {
		return "", ErrBadTarget
	}
	// args: <base...> <target> <command>. The command is a single argv element,
	// passed to the remote shell — no local shell is involved, so the target/
	// command cannot break out into the Pulse host's shell.
	args := append(append([]string{}, t.base[1:]...), target, command)
	cmd := exec.CommandContext(ctx, t.base[0], args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
