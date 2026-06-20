// Package dockerapi is a minimal client for the Docker Engine API over the
// host unix socket — just enough to launch a detached one-shot updater
// container. No docker CLI or heavy SDK required.
package dockerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	socket string
	http   *http.Client
}

// New returns a client talking to the daemon at socketPath (e.g.
// /var/run/docker.sock). It does not verify connectivity — call Ping.
func New(socketPath string) *Client {
	return &Client{
		socket: socketPath,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

// Ping reports whether the daemon is reachable.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/_ping", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker ping: %s", resp.Status)
	}
	return nil
}

// RunDetached pulls image (best-effort) then creates and starts a container
// with the given command and bind mounts, auto-removing on exit. Returns the
// container id.
func (c *Client) RunDetached(ctx context.Context, image string, cmd, binds []string) (string, error) {
	c.pull(ctx, image) // best-effort; create will fail if truly absent

	body, _ := json.Marshal(map[string]any{
		"Image": image,
		"Cmd":   cmd,
		"HostConfig": map[string]any{
			"Binds":      binds,
			"AutoRemove": true,
		},
	})
	resp, err := c.do(ctx, http.MethodPost, "/containers/create", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", apiError("create container", resp)
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", err
	}

	start, err := c.do(ctx, http.MethodPost, "/containers/"+created.ID+"/start", nil)
	if err != nil {
		return "", err
	}
	defer start.Body.Close()
	if start.StatusCode != http.StatusNoContent {
		return "", apiError("start container", start)
	}
	return created.ID, nil
}

// SelfImageRef inspects the given container (typically this process's own, by
// hostname) and returns its image's repository ref and pinned digest, e.g.
// ("ghcr.io/acme/pulse-backend", "sha256:abc…"). Empty digest means the image
// has no registry digest (e.g. built locally).
func (c *Client) SelfImageRef(ctx context.Context, container string) (repo, digest string, err error) {
	resp, err := c.do(ctx, http.MethodGet, "/containers/"+container+"/json", nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", apiError("inspect container", resp)
	}
	var ctr struct {
		Image string `json:"Image"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ctr); err != nil {
		return "", "", err
	}

	img, err := c.do(ctx, http.MethodGet, "/images/"+ctr.Image+"/json", nil)
	if err != nil {
		return "", "", err
	}
	defer img.Body.Close()
	if img.StatusCode != http.StatusOK {
		return "", "", apiError("inspect image", img)
	}
	var meta struct {
		RepoDigests []string `json:"RepoDigests"`
	}
	if err := json.NewDecoder(img.Body).Decode(&meta); err != nil {
		return "", "", err
	}
	for _, rd := range meta.RepoDigests {
		if i := strings.LastIndex(rd, "@"); i > 0 {
			return rd[:i], rd[i+1:], nil
		}
	}
	return "", "", nil
}

func (c *Client) pull(ctx context.Context, image string) {
	repo, tag := image, "latest"
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		repo, tag = image[:i], image[i+1:]
	}
	q := url.Values{"fromImage": {repo}, "tag": {tag}}
	resp, err := c.do(ctx, http.MethodPost, "/images/create?"+q.Encode(), nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // drain the pull progress stream
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

func apiError(action string, resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("%s: %s: %s", action, resp.Status, bytes.TrimSpace(b))
}
