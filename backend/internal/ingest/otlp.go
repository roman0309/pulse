package ingest

import (
	"encoding/json"
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
				out = append(out, RawLog{
					ServiceName: svc,
					Level:       mapSeverity(int32(lr.SeverityNumber), lr.SeverityText),
					Message:     anyString(lr.Body),
					Metadata:    attrsJSON(lr.Attributes),
					Timestamp:   unixNano(lr.TimeUnixNano),
				})
			}
		}
	}
	return out, nil
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
