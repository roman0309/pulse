// Package ingest decodes external telemetry formats (OTLP, Prometheus
// remote_write) into the platform's internal metric/log representation.
package ingest

import "time"

// RawMetric is a single decoded metric sample, before service resolution.
type RawMetric struct {
	ServiceName string
	MetricName  string
	Value       float64
	Timestamp   time.Time
}

// RawLog is a single decoded log record, before service resolution.
type RawLog struct {
	ServiceName string
	Level       string
	Message     string
	Metadata    string
	Timestamp   time.Time
}
