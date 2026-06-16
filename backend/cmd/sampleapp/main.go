// Command sampleapp is a self-contained, instrumented demo application that
// represents "the project's app running on a server". It serves real HTTP
// requests, drives a steady load against itself, measures its own RED metrics
// (Rate, Errors, Duration) and pushes them to Pulse as OTLP every interval —
// exactly how a real service reports application-level metrics into its project.
//
// Metrics emitted (canonical Pulse names):
//
//	request_count, request_rate, error_rate, latency_p50, latency_p95, latency_p99
//
// Env: PULSE_ENDPOINT, PULSE_KEY, PULSE_SERVICE, PULSE_INTERVAL, TARGET_RPS, PORT.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

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
		rps      int
		port     string
	}{
		endpoint: env("PULSE_ENDPOINT", "http://localhost:8080"),
		key:      env("PULSE_KEY", "pulse_demo_ingest_key"),
		service:  env("PULSE_SERVICE", "checkout-api"),
		interval: envDuration("PULSE_INTERVAL", 10*time.Second),
		rps:      envInt("TARGET_RPS", 25),
		port:     env("PORT", "9090"),
	}
	log.Printf("sampleapp: service=%q rps=%d endpoint=%s", cfg.service, cfg.rps, cfg.endpoint)

	col := &collector{start: time.Now()}

	// The application: a handler that does simulated work with realistic,
	// time-varying latency and an occasional error, measuring every request.
	mux := http.NewServeMux()
	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
		latency, failed := simulateWork()
		time.Sleep(latency)
		col.record(float64(latency.Milliseconds()), failed)
		if failed {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":" + cfg.port, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Load generator: drives steady traffic at the target rate against itself.
	go generateLoad("http://localhost:"+cfg.port+"/work", cfg.rps)

	// Reporter: snapshot RED metrics each interval and push to Pulse.
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()
	for range ticker.C {
		snap := col.snapshot(cfg.interval)
		if snap == nil {
			continue
		}
		body, err := buildOTLP(cfg.service, snap)
		if err != nil {
			log.Printf("encode: %v", err)
			continue
		}
		if err := push(context.Background(), cfg.endpoint+"/otlp/v1/metrics", cfg.key, body); err != nil {
			log.Printf("push: %v", err)
			continue
		}
		log.Printf("reported: count=%.0f rate=%.1f/s err=%.1f%% p95=%.0fms",
			snap["request_count"], snap["request_rate"], snap["error_rate"], snap["latency_p95"])
	}
}

// simulateWork returns a realistic latency and whether the request failed.
// A slow sine baseline plus jitter makes the charts lively; latency spikes
// raise the error probability, mimicking a degrading dependency.
func simulateWork() (time.Duration, bool) {
	phase := float64(time.Now().Unix()%120) / 120 * 2 * math.Pi
	base := 60 + 40*math.Sin(phase)        // 20..100ms baseline
	jitter := rand.Float64() * 50          // 0..50ms
	latency := base + jitter
	if rand.Float64() < 0.04 { // 4% tail: occasional slow request
		latency += 200 + rand.Float64()*400
	}
	// error probability rises with latency
	errProb := 0.01 + math.Max(0, (latency-150))/2000
	return time.Duration(latency) * time.Millisecond, rand.Float64() < errProb
}

func generateLoad(url string, rps int) {
	if rps <= 0 {
		rps = 1
	}
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()
	for range ticker.C {
		go func() {
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
}

// collector accumulates RED measurements over the current window.
type collector struct {
	mu        sync.Mutex
	count     int
	errors    int
	latencies []float64
	start     time.Time
}

func (c *collector) record(latencyMs float64, failed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	if failed {
		c.errors++
	}
	c.latencies = append(c.latencies, latencyMs)
}

// snapshot computes the RED metrics for the window and resets it.
func (c *collector) snapshot(window time.Duration) map[string]float64 {
	c.mu.Lock()
	count, errors, lats := c.count, c.errors, c.latencies
	c.count, c.errors, c.latencies = 0, 0, nil
	c.mu.Unlock()

	if count == 0 {
		return nil
	}
	sort.Float64s(lats)
	secs := window.Seconds()
	return map[string]float64{
		"request_count": float64(count),
		"request_rate":  float64(count) / secs,
		"error_rate":    float64(errors) / float64(count) * 100,
		"latency_p50":   percentile(lats, 50),
		"latency_p95":   percentile(lats, 95),
		"latency_p99":   percentile(lats, 99),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func buildOTLP(service string, metrics map[string]float64) ([]byte, error) {
	ts := uint64(time.Now().UnixNano())
	ms := make([]*mpb.Metric, 0, len(metrics))
	for name, v := range metrics {
		ms = append(ms, &mpb.Metric{
			Name: name,
			Data: &mpb.Metric_Gauge{Gauge: &mpb.Gauge{
				DataPoints: []*mpb.NumberDataPoint{{
					TimeUnixNano: ts,
					Value:        &mpb.NumberDataPoint_AsDouble{AsDouble: v},
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
			ScopeMetrics: []*mpb.ScopeMetrics{{Metrics: ms}},
		}},
	}
	return proto.Marshal(req)
}

func push(ctx context.Context, url, key string, body []byte) error {
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
		return fmt.Errorf("ingest rejected: %s", resp.Status)
	}
	return nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
