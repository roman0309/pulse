package ingest

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	lcol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	mcol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

// DecodeOTLPMetrics parses an OTLP/HTTP ExportMetricsServiceRequest (protobuf)
// into flat metric samples. Gauge and Sum metrics are supported (the common
// case for infra metrics); histograms are skipped for the MVP.
func DecodeOTLPMetrics(body []byte) ([]RawMetric, error) {
	var req mcol.ExportMetricsServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	var out []RawMetric
	for _, rm := range req.ResourceMetrics {
		svc := serviceName(rm.Resource)
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if _, ok := m.Data.(*mpb.Metric_Histogram); ok {
					out = append(out, histogramMetrics(m, svc)...)
					continue
				}
				for _, dp := range numberDataPoints(m) {
					out = append(out, RawMetric{
						ServiceName: svc,
						MetricName:  m.Name,
						Value:       numberValue(dp),
						Timestamp:   unixNano(dp.TimeUnixNano),
					})
				}
			}
		}
	}
	return out, nil
}

// DecodeOTLPLogs parses an OTLP/HTTP ExportLogsServiceRequest (protobuf).
func DecodeOTLPLogs(body []byte) ([]RawLog, error) {
	var req lcol.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	var out []RawLog
	for _, rl := range req.ResourceLogs {
		svc := serviceName(rl.Resource)
		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				traceID := ""
				if len(lr.TraceId) > 0 {
					traceID = hex.EncodeToString(lr.TraceId)
				}
				out = append(out, RawLog{
					ServiceName: svc,
					Level:       mapSeverity(int32(lr.SeverityNumber), lr.SeverityText),
					Message:     anyString(lr.Body),
					Metadata:    attrsJSON(lr.Attributes),
					TraceID:     traceID,
					Timestamp:   unixNano(lr.TimeUnixNano),
				})
			}
		}
	}
	return out, nil
}

// histogramMetrics converts an OTLP histogram into Pulse's canonical metrics.
// For request-duration histograms (e.g. from OpenTelemetry/Beyla) it derives
// latency_p50/p95/p99, request_count and error_rate. Use delta temporality on
// the sender so each export reflects the recent interval.
func histogramMetrics(m *mpb.Metric, svc string) []RawMetric {
	h, ok := m.Data.(*mpb.Metric_Histogram)
	if !ok || len(h.Histogram.DataPoints) == 0 {
		return nil
	}

	var bounds []float64
	var merged []uint64
	var total, errors uint64
	var ts uint64
	hasStatus := false

	for _, dp := range h.Histogram.DataPoints {
		if bounds == nil {
			bounds = dp.ExplicitBounds
			merged = make([]uint64, len(dp.BucketCounts))
		}
		if len(dp.BucketCounts) == len(merged) {
			for i, c := range dp.BucketCounts {
				merged[i] += c
			}
		}
		total += dp.Count
		if dp.TimeUnixNano > ts {
			ts = dp.TimeUnixNano
		}
		if code, ok := statusCode(dp.Attributes); ok {
			hasStatus = true
			if code >= 500 {
				errors += dp.Count
			}
		}
	}
	if total == 0 {
		return nil
	}

	scale := 1.0
	if m.Unit == "s" { // OTel/Beyla report durations in seconds → store ms
		scale = 1000
	}
	t := unixNano(ts)
	name := strings.ToLower(m.Name)
	isDuration := strings.Contains(name, "duration") ||
		strings.Contains(name, "latency") ||
		strings.Contains(name, "http.server.request")

	if !isDuration {
		base := sanitizeMetric(m.Name)
		return []RawMetric{{ServiceName: svc, MetricName: base + "_p95", Value: histQuantile(bounds, merged, 0.95) * scale, Timestamp: t}}
	}

	out := []RawMetric{
		{ServiceName: svc, MetricName: "latency_p50", Value: histQuantile(bounds, merged, 0.50) * scale, Timestamp: t},
		{ServiceName: svc, MetricName: "latency_p95", Value: histQuantile(bounds, merged, 0.95) * scale, Timestamp: t},
		{ServiceName: svc, MetricName: "latency_p99", Value: histQuantile(bounds, merged, 0.99) * scale, Timestamp: t},
		{ServiceName: svc, MetricName: "request_count", Value: float64(total), Timestamp: t},
	}
	if hasStatus {
		out = append(out, RawMetric{ServiceName: svc, MetricName: "error_rate", Value: float64(errors) / float64(total) * 100, Timestamp: t})
	}
	return out
}

// histQuantile estimates the q-quantile (0..1) from cumulative bucket counts
// with the given explicit upper bounds (len(counts) == len(bounds)+1).
func histQuantile(bounds []float64, counts []uint64, q float64) float64 {
	var total uint64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	rank := q * float64(total)
	var cum float64
	for i, c := range counts {
		prev := cum
		cum += float64(c)
		if cum < rank {
			continue
		}
		if i >= len(bounds) { // overflow bucket [last bound, +inf): no upper bound
			if len(bounds) == 0 {
				return 0
			}
			return bounds[len(bounds)-1]
		}
		lower := 0.0
		if i > 0 {
			lower = bounds[i-1]
		}
		upper := bounds[i]
		if c == 0 {
			return upper
		}
		return lower + (upper-lower)*((rank-prev)/float64(c))
	}
	return 0
}

func statusCode(attrs []*cpb.KeyValue) (int64, bool) {
	for _, kv := range attrs {
		if kv.Key == "http.response.status.code" || kv.Key == "http.status_code" {
			if v, ok := kv.Value.GetValue().(*cpb.AnyValue_IntValue); ok {
				return v.IntValue, true
			}
		}
	}
	return 0, false
}

func sanitizeMetric(s string) string {
	return strings.NewReplacer(".", "_", "-", "_", "/", "_").Replace(strings.ToLower(s))
}

func numberDataPoints(m *mpb.Metric) []*mpb.NumberDataPoint {
	switch d := m.Data.(type) {
	case *mpb.Metric_Gauge:
		return d.Gauge.DataPoints
	case *mpb.Metric_Sum:
		return d.Sum.DataPoints
	default:
		return nil
	}
}

func numberValue(dp *mpb.NumberDataPoint) float64 {
	switch v := dp.Value.(type) {
	case *mpb.NumberDataPoint_AsDouble:
		return v.AsDouble
	case *mpb.NumberDataPoint_AsInt:
		return float64(v.AsInt)
	default:
		return 0
	}
}

func serviceName(res *rpb.Resource) string {
	if res == nil {
		return "unknown"
	}
	for _, kv := range res.Attributes {
		if kv.Key == "service.name" {
			if s := anyString(kv.Value); s != "" {
				return s
			}
		}
	}
	return "unknown"
}

func anyString(v *cpb.AnyValue) string {
	if v == nil {
		return ""
	}
	switch x := v.Value.(type) {
	case *cpb.AnyValue_StringValue:
		return x.StringValue
	default:
		return ""
	}
}

func attrsJSON(kvs []*cpb.KeyValue) string {
	if len(kvs) == 0 {
		return "{}"
	}
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = anyString(kv.Value)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// mapSeverity maps OTLP severity numbers (1..24) to our level vocabulary.
func mapSeverity(num int32, text string) string {
	switch {
	case num >= 17: // ERROR / FATAL
		return "error"
	case num >= 13: // WARN
		return "warning"
	case num > 0: // TRACE / DEBUG / INFO
		return "info"
	}
	switch text {
	case "ERROR", "FATAL", "error", "fatal":
		return "error"
	case "WARN", "WARNING", "warn", "warning":
		return "warning"
	default:
		return "info"
	}
}

func unixNano(ts uint64) time.Time {
	if ts == 0 {
		return time.Now()
	}
	return time.Unix(0, int64(ts)).UTC()
}
