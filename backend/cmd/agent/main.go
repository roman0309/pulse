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
	"log"
	"net/http"
	"os"
	"time"

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
