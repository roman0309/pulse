package ingest

import (
	"errors"
	"math"
	"time"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/encoding/protowire"
)

// DecodePromRemoteWrite decodes a Prometheus remote_write request body
// (snappy-compressed protobuf WriteRequest) into flat metric samples.
//
// The WriteRequest schema is small and stable, so it is decoded directly with
// the protobuf wire reader instead of pulling in the entire Prometheus module:
//
//	WriteRequest { repeated TimeSeries timeseries = 1; }
//	TimeSeries   { repeated Label labels = 1; repeated Sample samples = 2; }
//	Label        { string name = 1; string value = 2; }
//	Sample       { double value = 1; int64 timestamp = 2; }   // ms
func DecodePromRemoteWrite(body []byte) ([]RawMetric, error) {
	raw, err := snappy.Decode(nil, body)
	if err != nil {
		return nil, err
	}
	var out []RawMetric
	b := raw
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, errParse
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			msg, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, errParse
			}
			b = b[n:]
			samples, err := parseTimeSeries(msg)
			if err != nil {
				return nil, err
			}
			out = append(out, samples...)
		} else {
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, errParse
			}
			b = b[n:]
		}
	}
	return out, nil
}

var errParse = errors.New("invalid remote_write protobuf")

func parseTimeSeries(b []byte) ([]RawMetric, error) {
	labels := map[string]string{}
	type sample struct {
		v  float64
		ts int64
	}
	var samples []sample

	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, errParse
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType: // Label
			msg, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, errParse
			}
			b = b[n:]
			name, val, err := parseLabel(msg)
			if err != nil {
				return nil, err
			}
			labels[name] = val
		case num == 2 && typ == protowire.BytesType: // Sample
			msg, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, errParse
			}
			b = b[n:]
			v, ts, err := parseSample(msg)
			if err != nil {
				return nil, err
			}
			samples = append(samples, sample{v, ts})
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, errParse
			}
			b = b[n:]
		}
	}

	metricName := labels["__name__"]
	if metricName == "" {
		return nil, nil
	}
	svc := firstNonEmpty(labels["service"], labels["job"], labels["instance"], "unknown")

	out := make([]RawMetric, 0, len(samples))
	for _, s := range samples {
		ts := time.Now()
		if s.ts > 0 {
			ts = time.UnixMilli(s.ts).UTC()
		}
		out = append(out, RawMetric{
			ServiceName: svc,
			MetricName:  metricName,
			Value:       s.v,
			Timestamp:   ts,
		})
	}
	return out, nil
}

func parseLabel(b []byte) (name, value string, err error) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return "", "", errParse
		}
		b = b[n:]
		if typ == protowire.BytesType {
			s, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return "", "", errParse
			}
			b = b[n:]
			if num == 1 {
				name = string(s)
			} else if num == 2 {
				value = string(s)
			}
		} else {
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return "", "", errParse
			}
			b = b[n:]
		}
	}
	return name, value, nil
}

func parseSample(b []byte) (value float64, ts int64, err error) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return 0, 0, errParse
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.Fixed64Type:
			bits, n := protowire.ConsumeFixed64(b)
			if n < 0 {
				return 0, 0, errParse
			}
			b = b[n:]
			value = math.Float64frombits(bits)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return 0, 0, errParse
			}
			b = b[n:]
			ts = int64(v)
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return 0, 0, errParse
			}
			b = b[n:]
		}
	}
	return value, ts, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
