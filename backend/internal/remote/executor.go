// Package remote runs commands on managed servers over SSH using stored
// (encrypted) credentials.
package remote

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

// Conn describes how to reach a server over SSH. Exactly one of Password or
// PrivateKey is used (PrivateKey takes precedence).
type Conn struct {
	Host       string
	Port       int
	User       string
	Password   string
	PrivateKey string
	// KnownHostKey, if set, is the pinned host-key fingerprint that the server
	// must present (trust-on-first-use). Empty = accept and pin on first connect.
	KnownHostKey string
}

// Executor runs a shell command on a remote host. It returns the command output
// and the server's host-key fingerprint (for TOFU pinning by the caller).
type Executor interface {
	Run(ctx context.Context, conn Conn, command string) (output string, hostKey string, err error)
}

var (
	ErrBadConn        = errors.New("invalid ssh connection (host/user/credentials required)")
	ErrHostKeyChanged = errors.New("ssh host key changed — refusing to connect (possible MITM)")
)

// SSHExecutor connects with golang.org/x/crypto/ssh.
type SSHExecutor struct {
	DialTimeout time.Duration
}

func NewSSH() *SSHExecutor { return &SSHExecutor{DialTimeout: 10 * time.Second} }

func (e *SSHExecutor) Run(ctx context.Context, conn Conn, command string) (string, string, error) {
	if conn.Host == "" || conn.User == "" {
		return "", "", ErrBadConn
	}
	var auth []ssh.AuthMethod
	switch {
	case conn.PrivateKey != "":
		signer, err := ssh.ParsePrivateKey([]byte(conn.PrivateKey))
		if err != nil {
			return "", "", fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	case conn.Password != "":
		auth = append(auth, ssh.Password(conn.Password))
	default:
		return "", "", ErrBadConn
	}

	port := conn.Port
	if port == 0 {
		port = 22
	}

	// Trust-on-first-use host-key check: capture the presented key; if a key was
	// already pinned, require it to match.
	var observed string
	hostKeyCB := func(_ string, _ net.Addr, key ssh.PublicKey) error {
		observed = ssh.FingerprintSHA256(key)
		if conn.KnownHostKey != "" && observed != conn.KnownHostKey {
			return ErrHostKeyChanged
		}
		return nil
	}

	cfg := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCB,
		Timeout:         e.DialTimeout,
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(conn.Host, strconv.Itoa(port)), cfg)
	if err != nil {
		return "", observed, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", observed, err
	}
	defer session.Close()

	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := session.CombinedOutput(command)
		done <- result{out, err}
	}()

	select {
	case r := <-done:
		return string(r.out), observed, r.err
	case <-ctx.Done():
		_ = session.Close()
		return "", observed, ctx.Err()
	}
}
