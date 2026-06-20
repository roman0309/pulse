package ingest

import (
	"encoding/hex"

	tcol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tpb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// DecodeOTLPTraces parses an OTLP/HTTP ExportTraceServiceRequest (protobuf)
// into flat spans. Beyla and standard OTel SDKs export to /otlp/v1/traces.
func DecodeOTLPTraces(body []byte) ([]RawSpan, error) {
	var req tcol.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	var out []RawSpan
	for _, rs := range req.ResourceSpans {
		svc := serviceName(rs.Resource)
		for _, ss := range rs.ScopeSpans {
			for _, sp := range ss.Spans {
				start := unixNano(sp.StartTimeUnixNano)
				dur := 0.0
				if sp.EndTimeUnixNano > sp.StartTimeUnixNano {
					dur = float64(sp.EndTimeUnixNano-sp.StartTimeUnixNano) / 1e6
				}
				out = append(out, RawSpan{
					TraceID:     hex.EncodeToString(sp.TraceId),
					SpanID:      hex.EncodeToString(sp.SpanId),
					ParentID:    hex.EncodeToString(sp.ParentSpanId),
					ServiceName: svc,
					Name:        sp.Name,
					Kind:        spanKind(sp.Kind),
					StatusCode:  spanStatus(sp.Status),
					StartTime:   start,
					DurationMS:  dur,
					Attributes:  attrsJSON(sp.Attributes),
				})
			}
		}
	}
	return out, nil
}

func spanKind(k tpb.Span_SpanKind) string {
	switch k {
	case tpb.Span_SPAN_KIND_SERVER:
		return "server"
	case tpb.Span_SPAN_KIND_CLIENT:
		return "client"
	case tpb.Span_SPAN_KIND_PRODUCER:
		return "producer"
	case tpb.Span_SPAN_KIND_CONSUMER:
		return "consumer"
	default:
		return "internal"
	}
}

func spanStatus(s *tpb.Status) string {
	if s == nil {
		return "unset"
	}
	switch s.Code {
	case tpb.Status_STATUS_CODE_OK:
		return "ok"
	case tpb.Status_STATUS_CODE_ERROR:
		return "error"
	default:
		return "unset"
	}
}
