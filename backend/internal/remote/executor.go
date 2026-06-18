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
}

// Executor runs a shell command on a remote host.
type Executor interface {
	Run(ctx context.Context, conn Conn, command string) (output string, err error)
}

var ErrBadConn = errors.New("invalid ssh connection (host/user/credentials required)")

// SSHExecutor connects with golang.org/x/crypto/ssh.
type SSHExecutor struct {
	DialTimeout time.Duration
}

func NewSSH() *SSHExecutor { return &SSHExecutor{DialTimeout: 10 * time.Second} }

func (e *SSHExecutor) Run(ctx context.Context, conn Conn, command string) (string, error) {
	if conn.Host == "" || conn.User == "" {
		return "", ErrBadConn
	}
	var auth []ssh.AuthMethod
	switch {
	case conn.PrivateKey != "":
		signer, err := ssh.ParsePrivateKey([]byte(conn.PrivateKey))
		if err != nil {
			return "", fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	case conn.Password != "":
		auth = append(auth, ssh.Password(conn.Password))
	default:
		return "", ErrBadConn
	}

	port := conn.Port
	if port == 0 {
		port = 22
	}
	cfg := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: TOFU/known_hosts for stricter checking
		Timeout:         e.DialTimeout,
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(conn.Host, strconv.Itoa(port)), cfg)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
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
		return string(r.out), r.err
	case <-ctx.Done():
		_ = session.Close()
		return "", ctx.Err()
	}
}
