// Command agent is the Pulse host agent. It samples real system metrics
// (CPU, memory, disk, load) on the machine it runs on and pushes them to a
// Pulse backend as OTLP, authenticated by a per-project ingest key.
//
// Deploy it on any server you want to monitor:
//
//	PULSE_ENDPOINT=https://pulse.example.com \
//	PULSE_KEY=<project ingest key> \
//	PULSE_SERVICE=payment-api \
//	./agent
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	mcol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	cfg := struct {
		endpoint string
		key      string
		service  string
		interval time.Duration
	}{
		endpoint: env("PULSE_ENDPOINT", "http://localhost:8080"),
		key:      env("PULSE_KEY", "pulse_demo_ingest_key"),
		service:  env("PULSE_SERVICE", hostname()),
		interval: envDuration("PULSE_INTERVAL", 10*time.Second),
	}

	log.Printf("pulse-agent: service=%q endpoint=%s interval=%s", cfg.service, cfg.endpoint, cfg.interval)

	// Control channel: outbound WS to Pulse for remote commands (install Beyla,
	// status, remove). Best-effort — metrics keep flowing even if it can't connect.
	go runControl(cfg.endpoint, cfg.key, cfg.service)

	// Ship Docker container logs (best-effort; needs the mounted docker socket).
	go runLogs(cfg.endpoint, cfg.key)

	// Record deployments when containers are (re)created.
	go runDeployments(cfg.endpoint, cfg.key)

	// Prime the CPU counter so the first real reading is meaningful.
	_, _ = cpu.Percent(0, false)
	time.Sleep(time.Second)

	ctx := context.Background()
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	send := func() {
		samples := collect()
		if len(samples) == 0 {
			return
		}
		body, err := buildOTLP(cfg.service, samples)
		if err != nil {
			log.Printf("encode error: %v", err)
			return
		}
		if err := post(ctx, cfg.endpoint+"/otlp/v1/metrics", cfg.key, body); err != nil {
			log.Printf("push error: %v", err)
			return
		}
		log.Printf("pushed %d metrics", len(samples))
	}

	send() // immediate first sample
	for range ticker.C {
		send()
	}
}

type sample struct {
	name  string
	value float64
}

// collect reads the current system metrics, using canonical Pulse metric names.
func collect() []sample {
	var out []sample
	now := func(name string, v float64) { out = append(out, sample{name, v}) }

	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		now("cpu_usage", pct[0])
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		now("memory_usage", vm.UsedPercent)
	}
	if du, err := disk.Usage("/"); err == nil {
		now("disk_usage", du.UsedPercent)
	}
	if la, err := load.Avg(); err == nil {
		now("load_1m", la.Load1)
	}
	return out
}

func buildOTLP(service string, samples []sample) ([]byte, error) {
	ts := uint64(time.Now().UnixNano())
	metrics := make([]*mpb.Metric, 0, len(samples))
	for _, s := range samples {
		metrics = append(metrics, &mpb.Metric{
			Name: s.name,
			Data: &mpb.Metric_Gauge{Gauge: &mpb.Gauge{
				DataPoints: []*mpb.NumberDataPoint{{
					TimeUnixNano: ts,
					Value:        &mpb.NumberDataPoint_AsDouble{AsDouble: s.value},
				}},
			}},
		})
	}
	req := &mcol.ExportMetricsServiceRequest{
		ResourceMetrics: []*mpb.ResourceMetrics{{
			Resource: &rpb.Resource{Attributes: []*cpb.KeyValue{{
				Key:   "service.name",
				Value: &cpb.AnyValue{Value: &cpb.AnyValue_StringValue{StringValue: service}},
			}}},
			ScopeMetrics: []*mpb.ScopeMetrics{{Metrics: metrics}},
		}},
	}
	return proto.Marshal(req)
}

func post(ctx context.Context, url, key string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("X-Pulse-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return &httpError{resp.StatusCode}
	}
	return nil
}

type httpError struct{ code int }

func (e *httpError) Error() string { return "unexpected status " + http.StatusText(e.code) }

func postJSON(ctx context.Context, url, key string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pulse-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return &httpError{resp.StatusCode}
	}
	return nil
}

// ---------- Container log collection ----------

type logLine struct {
	Service   string `json:"service"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// traceIDRe finds a trace id logged inline, e.g. trace_id=abc, traceID:"abc".
var traceIDRe = regexp.MustCompile(`(?i)trace[_-]?id["']?\s*[:=]\s*["']?([0-9a-f]{16,32})`)

func parseTraceID(msg string) string {
	if m := traceIDRe.FindStringSubmatch(msg); len(m) == 2 {
		return strings.ToLower(m[1])
	}
	return ""
}

// runLogs ships Docker container logs to Pulse. Best-effort: needs the docker
// socket (mounted) and CLI (present in the agent image). Each container's logs
// land under a service named after the container.
func runLogs(endpoint, key string) {
	if _, err := dockerOut("version", "--format", "{{.Server.Version}}"); err != nil {
		log.Printf("logs: docker unavailable, skipping container log collection")
		return
	}
	log.Printf("logs: collecting container logs")
	cursors := map[string]time.Time{}
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		collectLogs(endpoint, key, cursors)
		<-t.C
	}
}

func collectLogs(endpoint, key string, cursors map[string]time.Time) {
	out, err := dockerOut("ps", "--format", "{{.Names}}")
	if err != nil {
		return
	}
	var batch []logLine
	for _, name := range strings.Fields(out) {
		if name == "pulse-agent" {
			continue // don't ship our own logs
		}
		since := cursors[name]
		if since.IsZero() {
			since = time.Now().Add(-1 * time.Minute)
		}
		raw, err := dockerOut("logs", "--since", since.Format(time.RFC3339Nano), "--timestamps", "--tail", "500", name)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(raw, "\n") {
			ts, msg, ok := parseLogLine(line)
			if !ok || !ts.After(since) {
				continue
			}
			batch = append(batch, logLine{
				Service:   name,
				Level:     detectLevel(msg),
				Message:   truncate(msg, 4000),
				TraceID:   parseTraceID(msg),
				Timestamp: ts.UTC().Format(time.RFC3339Nano),
			})
			if ts.After(cursors[name]) {
				cursors[name] = ts
			}
		}
		if len(batch) >= 1000 {
			break
		}
	}
	if len(batch) == 0 {
		return
	}
	body, err := json.Marshal(map[string]any{"logs": batch})
	if err != nil {
		return
	}
	if err := postJSON(context.Background(), endpoint+"/api/v1/ingest/logs", key, body); err != nil {
		log.Printf("logs push error: %v", err)
		return
	}
	log.Printf("pushed %d logs", len(batch))
}

// parseLogLine splits a `docker logs --timestamps` line into its RFC3339
// timestamp and the message.
func parseLogLine(line string) (time.Time, string, bool) {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return time.Time{}, "", false
	}
	sp := strings.IndexByte(line, ' ')
	if sp < 0 {
		return time.Time{}, "", false
	}
	ts, err := time.Parse(time.RFC3339Nano, line[:sp])
	if err != nil {
		return time.Time{}, "", false
	}
	return ts, line[sp+1:], true
}

// ---------- Deployment detection ----------

type deployItem struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// runDeployments watches container creation times; when a container is
// recreated (a deploy), it records a deployment. The first poll just snapshots,
// so existing containers aren't reported as deployments on agent start.
func runDeployments(endpoint, key string) {
	if _, err := dockerOut("version", "--format", "{{.Server.Version}}"); err != nil {
		return
	}
	seen := map[string]string{} // container name -> CreatedAt
	first := true
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		out, err := dockerOut("ps", "--format", "{{.Names}}\t{{.CreatedAt}}\t{{.Image}}")
		if err == nil {
			var batch []deployItem
			for _, line := range strings.Split(out, "\n") {
				parts := strings.Split(strings.TrimRight(line, "\r"), "\t")
				if len(parts) != 3 {
					continue
				}
				name, created, image := parts[0], parts[1], parts[2]
				if name == "pulse-agent" {
					continue
				}
				if prev, ok := seen[name]; ok && prev != created && !first {
					batch = append(batch, deployItem{Service: name, Version: image, Status: "success"})
				}
				seen[name] = created
			}
			if len(batch) > 0 {
				if body, e := json.Marshal(map[string]any{"deployments": batch}); e == nil {
					if err := postJSON(context.Background(), endpoint+"/api/v1/ingest/deployments", key, body); err != nil {
						log.Printf("deployments push error: %v", err)
					} else {
						log.Printf("recorded %d deployment(s)", len(batch))
					}
				}
			}
			first = false
		}
		<-t.C
	}
}

func detectLevel(msg string) string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "error") || strings.Contains(m, "fatal") || strings.Contains(m, "panic") || strings.Contains(m, "[err"):
		return "error"
	case strings.Contains(m, "warn"):
		return "warning"
	default:
		return "info"
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---------- Control channel ----------

type ctrlCommand struct {
	ID   string            `json:"id"`
	Cmd  string            `json:"cmd"`
	Args map[string]string `json:"args"`
}

type ctrlReply struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

// runControl maintains an outbound WebSocket to Pulse and executes commands.
func runControl(endpoint, key, service string) {
	wsURL := toWS(endpoint) + "/api/v1/agent/connect?agent=" + url.QueryEscape(service)
	hdr := http.Header{"X-Pulse-Key": []string{key}}
	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("control channel connected")
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var cmd ctrlCommand
			if json.Unmarshal(msg, &cmd) != nil {
				continue
			}
			reply := handleCommand(cmd, endpoint, key, service)
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_ = conn.WriteJSON(reply)
		}
		conn.Close()
		time.Sleep(3 * time.Second)
	}
}

func handleCommand(cmd ctrlCommand, endpoint, key, service string) ctrlReply {
	switch cmd.Cmd {
	case "ping":
		return ctrlReply{cmd.ID, true, "pong"}
	case "status":
		out, err := dockerOut("ps", "--filter", "name=pulse-beyla", "--format", "{{.Names}} {{.Status}}")
		return ctrlReply{cmd.ID, err == nil, strOr(out, "no app-metrics agent running")}
	case "install_beyla":
		port := cmd.Args["port"]
		if port == "" {
			port = "8080"
		}
		_, _ = dockerOut("rm", "-f", "pulse-beyla")
		out, err := dockerOut("run", "-d", "--name", "pulse-beyla", "--restart", "unless-stopped",
			"--privileged", "--pid=host",
			"-e", "BEYLA_OPEN_PORT="+port,
			"-e", "OTEL_SERVICE_NAME="+service,
			"-e", "OTEL_EXPORTER_OTLP_ENDPOINT="+endpoint+"/otlp",
			"-e", "OTEL_EXPORTER_OTLP_HEADERS=X-Pulse-Key="+key,
			"-e", "OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=delta",
			"grafana/beyla:latest")
		return ctrlReply{cmd.ID, err == nil, out}
	case "remove":
		out, err := dockerOut("rm", "-f", "pulse-beyla")
		return ctrlReply{cmd.ID, err == nil, strOr(out, "removed")}
	default:
		return ctrlReply{cmd.ID, false, "unknown command: " + cmd.Cmd}
	}
}

func dockerOut(args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil && s == "" {
		s = err.Error()
	}
	return s, err
}

func toWS(endpoint string) string {
	if strings.HasPrefix(endpoint, "https://") {
		return "wss://" + strings.TrimPrefix(endpoint, "https://")
	}
	return "ws://" + strings.TrimPrefix(endpoint, "http://")
}

func strOr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func hostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return "host-" + h
	}
	return "host"
}
