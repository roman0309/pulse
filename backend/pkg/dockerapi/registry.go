package dockerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var registryHTTP = &http.Client{Timeout: 10 * time.Second}

const manifestAccept = "application/vnd.oci.image.index.v1+json," +
	"application/vnd.oci.image.manifest.v1+json," +
	"application/vnd.docker.distribution.manifest.list.v2+json," +
	"application/vnd.docker.distribution.manifest.v2+json"

// LatestManifestDigest returns the registry's current digest for repoRef's
// "latest" tag, e.g. repoRef="ghcr.io/acme/pulse-backend". Works for public
// images via an anonymous pull token (GHCR-style token endpoint).
func LatestManifestDigest(ctx context.Context, repoRef string) (string, error) {
	host, repo, ok := splitRepo(repoRef)
	if !ok {
		return "", fmt.Errorf("unrecognised image ref %q", repoRef)
	}

	token, err := pullToken(ctx, host, repo)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead,
		fmt.Sprintf("https://%s/v2/%s/manifests/latest", host, repo), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", manifestAccept)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := registryHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry manifest: %s", resp.Status)
	}
	return resp.Header.Get("Docker-Content-Digest"), nil
}

func pullToken(ctx context.Context, host, repo string) (string, error) {
	u := fmt.Sprintf("https://%s/token?service=%s&scope=%s",
		host, url.QueryEscape(host), url.QueryEscape("repository:"+repo+":pull"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := registryHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Some registries don't require auth for public pulls.
		return "", nil
	}
	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Token != "" {
		return body.Token, nil
	}
	return body.AccessToken, nil
}

// splitRepo separates "ghcr.io/acme/pulse-backend[:tag]" into host + repo path.
func splitRepo(ref string) (host, repo string, ok bool) {
	// drop any tag/digest
	if i := strings.LastIndex(ref, "@"); i > 0 {
		ref = ref[:i]
	}
	slash := strings.IndexByte(ref, '/')
	if slash < 0 {
		return "", "", false
	}
	host = ref[:slash]
	rest := ref[slash+1:]
	if i := strings.LastIndex(rest, ":"); i > strings.LastIndex(rest, "/") {
		rest = rest[:i]
	}
	if !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return "", "", false // not a registry host (would be a Docker Hub short name)
	}
	return host, rest, true
}
