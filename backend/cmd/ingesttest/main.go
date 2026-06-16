// Command ingesttest sends a real OTLP metrics payload and a Prometheus
// remote_write payload to a running backend, to verify the ingestion pipeline
// end-to-end. Usage:
//
//	go run ./cmd/ingesttest http://localhost:8080 pulse_demo_ingest_key
package main

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	mcol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/golang/snappy"
)

func main() {
	base := arg(1, "http://localhost:8080")
	key := arg(2, "pulse_demo_ingest_key")

	now := uint64(time.Now().UnixNano())

	// ---- OTLP metrics: gauge "otlp_test_cpu" for service "otlp-demo" ----
	otlpReq := &mcol.ExportMetricsServiceRequest{
		ResourceMetrics: []*mpb.ResourceMetrics{{
			Resource: &rpb.Resource{Attributes: []*cpb.KeyValue{{
				Key:   "service.name",
				Value: &cpb.AnyValue{Value: &cpb.AnyValue_StringValue{StringValue: "otlp-demo"}},
			}}},
			ScopeMetrics: []*mpb.ScopeMetrics{{
				Metrics: []*mpb.Metric{{
					Name: "otlp_test_cpu",
					Data: &mpb.Metric_Gauge{Gauge: &mpb.Gauge{
						DataPoints: []*mpb.NumberDataPoint{{
							TimeUnixNano: now,
							Value:        &mpb.NumberDataPoint_AsDouble{AsDouble: 42.5},
						}},
					}},
				}},
			}},
		}},
	}
	otlpBody, err := proto.Marshal(otlpReq)
	must(err)
	post(base+"/otlp/v1/metrics", key, "application/x-protobuf", "", otlpBody)

	// ---- Prometheus remote_write: "prom_test_qps" for service "prom-demo" ----
	wr := buildWriteRequest("prom_test_qps", "prom-demo", 123.0, time.Now().UnixMilli())
	post(base+"/api/v1/prom/write", key, "application/x-protobuf", "snappy", snappy.Encode(nil, wr))

	// ---- Optional: send OTLP through the collector (arg 3 = collector base) ----
	// The collector injects the ingest key, so none is sent here.
	if collector := arg(3, ""); collector != "" {
		colReq := &mcol.ExportMetricsServiceRequest{
			ResourceMetrics: []*mpb.ResourceMetrics{{
				Resource: &rpb.Resource{Attributes: []*cpb.KeyValue{{
					Key:   "service.name",
					Value: &cpb.AnyValue{Value: &cpb.AnyValue_StringValue{StringValue: "collector-demo"}},
				}}},
				ScopeMetrics: []*mpb.ScopeMetrics{{
					Metrics: []*mpb.Metric{{
						Name: "collector_test_mem",
						Data: &mpb.Metric_Gauge{Gauge: &mpb.Gauge{
							DataPoints: []*mpb.NumberDataPoint{{
								TimeUnixNano: uint64(time.Now().UnixNano()),
								Value:        &mpb.NumberDataPoint_AsDouble{AsDouble: 77.0},
							}},
						}},
					}},
				}},
			}},
		}
		colBody, err := proto.Marshal(colReq)
		must(err)
		post(collector+"/v1/metrics", "", "application/x-protobuf", "", colBody)
	}

	fmt.Println("done")
}

// buildWriteRequest hand-encodes a minimal remote_write WriteRequest.
func buildWriteRequest(metric, service string, value float64, tsMillis int64) []byte {
	label := func(name, val string) []byte {
		var b []byte
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, []byte(name))
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendBytes(b, []byte(val))
		return b
	}
	var sample []byte
	sample = protowire.AppendTag(sample, 1, protowire.Fixed64Type)
	sample = protowire.AppendFixed64(sample, math.Float64bits(value))
	sample = protowire.AppendTag(sample, 2, protowire.VarintType)
	sample = protowire.AppendVarint(sample, uint64(tsMillis))

	var ts []byte
	for _, l := range [][2]string{{"__name__", metric}, {"service", service}} {
		lb := label(l[0], l[1])
		ts = protowire.AppendTag(ts, 1, protowire.BytesType)
		ts = protowire.AppendBytes(ts, lb)
	}
	ts = protowire.AppendTag(ts, 2, protowire.BytesType)
	ts = protowire.AppendBytes(ts, sample)

	var wr []byte
	wr = protowire.AppendTag(wr, 1, protowire.BytesType)
	wr = protowire.AppendBytes(wr, ts)
	return wr
}

func post(url, key, contentType, encoding string, body []byte) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	must(err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Pulse-Key", key)
	if encoding != "" {
		req.Header.Set("Content-Encoding", encoding)
	}
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	fmt.Printf("POST %s -> %d %s\n", url, resp.StatusCode, buf.String())
}

func arg(i int, def string) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return def
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
