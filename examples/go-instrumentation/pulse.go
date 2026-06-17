// Package pulse is a drop-in RED-metrics reporter for a Go HTTP service.
// Copy this file into your app, then:
//
//	pulse.Start("http://100.124.46.68:8080", "<your ingest key>", "messenger")
//	http.ListenAndServe(addr, pulse.Middleware(mux))
//
// It measures every request and pushes request_count / request_rate /
// error_rate / latency_p50 / latency_p95 / latency_p99 to Pulse every 10s via
// the JSON ingest endpoint — no extra dependencies.
package pulse

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"
)

const interval = 10 * time.Second

var c collector

type collector struct {
	mu        sync.Mutex
	count     int
	errors    int
	latencies []float64
}

func (col *collector) record(ms float64, isErr bool) {
	col.mu.Lock()
	defer col.mu.Unlock()
	col.count++
	if isErr {
		col.errors++
	}
	col.latencies = append(col.latencies, ms)
}

// Middleware wraps an http.Handler and records latency + errors per request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		c.record(float64(time.Since(start).Milliseconds()), sr.status >= 500)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Record lets non-HTTP code (e.g. a gRPC interceptor) report a handled call.
func Record(latencyMs float64, failed bool) { c.record(latencyMs, failed) }

// Start launches the background reporter.
func Start(endpoint, key, service string) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			report(endpoint, key, service)
		}
	}()
}

func report(endpoint, key, service string) {
	c.mu.Lock()
	count, errs, lats := c.count, c.errors, c.latencies
	c.count, c.errors, c.latencies = 0, 0, nil
	c.mu.Unlock()
	if count == 0 {
		return
	}
	sort.Float64s(lats)
	point := func(metric string, value float64) map[string]any {
		return map[string]any{"service": service, "metric": metric, "value": value}
	}
	body, _ := json.Marshal(map[string]any{"points": []map[string]any{
		point("request_count", float64(count)),
		point("request_rate", float64(count)/interval.Seconds()),
		point("error_rate", float64(errs)/float64(count)*100),
		point("latency_p50", pct(lats, 50)),
		point("latency_p95", pct(lats, 95)),
		point("latency_p99", pct(lats, 99)),
	}})

	req, err := http.NewRequest(http.MethodPost, endpoint+"/api/v1/ingest/metrics", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pulse-Key", key)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

func pct(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}
