package ingest

import (
	"testing"

	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func TestHistQuantile(t *testing.T) {
	// bounds: <=0.1, <=0.2, <=0.5, >0.5 ; counts per bucket
	bounds := []float64{0.1, 0.2, 0.5}
	counts := []uint64{50, 30, 15, 5} // total 100
	p50 := histQuantile(bounds, counts, 0.50)
	if p50 < 0 || p50 > 0.1 { // 50th falls in first bucket [0,0.1]
		t.Fatalf("p50 expected within [0,0.1], got %v", p50)
	}
	p95 := histQuantile(bounds, counts, 0.95)
	if p95 < 0.2 || p95 > 0.5 { // 95th falls in third bucket (0.2,0.5]
		t.Fatalf("p95 expected within (0.2,0.5], got %v", p95)
	}
}

func TestHistogramMetricsRED(t *testing.T) {
	mkDP := func(status int64, buckets []uint64) *mpb.HistogramDataPoint {
		var total uint64
		for _, c := range buckets {
			total += c
		}
		return &mpb.HistogramDataPoint{
			Count:          total,
			ExplicitBounds: []float64{0.1, 0.2, 0.5},
			BucketCounts:   buckets,
			TimeUnixNano:   1,
			Attributes: []*cpb.KeyValue{{
				Key:   "http.response.status.code",
				Value: &cpb.AnyValue{Value: &cpb.AnyValue_IntValue{IntValue: status}},
			}},
		}
	}
	m := &mpb.Metric{
		Name: "http.server.request.duration",
		Unit: "s",
		Data: &mpb.Metric_Histogram{Histogram: &mpb.Histogram{
			DataPoints: []*mpb.HistogramDataPoint{
				mkDP(200, []uint64{80, 10, 5, 0}),  // 95 ok requests
				mkDP(500, []uint64{1, 1, 2, 1}),    // 5 errors
			},
		}},
	}

	got := map[string]float64{}
	for _, rm := range histogramMetrics(m, "messenger") {
		got[rm.MetricName] = rm.Value
	}

	if _, ok := got["latency_p95"]; !ok {
		t.Fatal("expected latency_p95")
	}
	if got["latency_p95"] < got["latency_p50"] {
		t.Fatalf("p95 (%v) should be >= p50 (%v)", got["latency_p95"], got["latency_p50"])
	}
	// unit was seconds → values scaled to ms (bounds <=500ms)
	if got["latency_p95"] > 500 || got["latency_p95"] <= 0 {
		t.Fatalf("latency_p95 ms out of range: %v", got["latency_p95"])
	}
	if got["request_count"] != 100 {
		t.Fatalf("request_count expected 100, got %v", got["request_count"])
	}
	if got["error_rate"] < 4.9 || got["error_rate"] > 5.1 { // 5/100
		t.Fatalf("error_rate expected ~5, got %v", got["error_rate"])
	}
}
