package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/ingest"
	"github.com/acme/observability/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// anonService reports whether a telemetry stream carries no usable service
// identity. Beyla emits "unknown" for processes it can't name; we drop those
// rather than pollute the project with a phantom "unknown" service.
func anonService(name string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	return n == "" || n == "unknown"
}

const maxIngestBody = 16 << 20 // 16 MiB

// IngestHandler accepts OTLP and Prometheus remote_write telemetry. Requests
// are authenticated by an ingest key (resolved to a project by middleware).
type IngestHandler struct {
	core *services.CoreService
}

func NewIngestHandler(core *services.CoreService) *IngestHandler {
	return &IngestHandler{core: core}
}

// OTLPMetrics handles POST /otlp/v1/metrics
func (h *IngestHandler) OTLPMetrics(c *gin.Context) {
	body, ok := readBody(c)
	if !ok {
		return
	}
	raw, err := ingest.DecodeOTLPMetrics(body)
	if err != nil {
		badRequest(c, err)
		return
	}
	h.writeMetrics(c, raw)
}

// OTLPLogs handles POST /otlp/v1/logs
func (h *IngestHandler) OTLPLogs(c *gin.Context) {
	body, ok := readBody(c)
	if !ok {
		return
	}
	raw, err := ingest.DecodeOTLPLogs(body)
	if err != nil {
		badRequest(c, err)
		return
	}
	projectID := middleware.IngestProjectID(c)
	cache := newServiceCache(h.core, projectID, middleware.IngestKeyID(c))
	logs := make([]entities.LogEntry, 0, len(raw))
	for _, l := range raw {
		if anonService(l.ServiceName) {
			continue
		}
		sid, name := cache.resolve(c, l.ServiceName)
		logs = append(logs, entities.LogEntry{
			ProjectID:   projectID.String(),
			ServiceID:   sid.String(),
			ServiceName: name,
			Level:       l.Level,
			Message:     l.Message,
			Metadata:    l.Metadata,
			Timestamp:   l.Timestamp,
		})
	}
	if err := h.core.IngestLogs(c.Request.Context(), logs); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(logs)})
}

// OTLPTraces handles POST /otlp/v1/traces — stores spans for the trace view.
func (h *IngestHandler) OTLPTraces(c *gin.Context) {
	body, ok := readBody(c)
	if !ok {
		return
	}
	raw, err := ingest.DecodeOTLPTraces(body)
	if err != nil {
		badRequest(c, err)
		return
	}
	projectID := middleware.IngestProjectID(c)
	spans := make([]entities.Span, 0, len(raw))
	for _, s := range raw {
		if anonService(s.ServiceName) {
			continue
		}
		spans = append(spans, entities.Span{
			ProjectID:   projectID.String(),
			TraceID:     s.TraceID,
			SpanID:      s.SpanID,
			ParentID:    s.ParentID,
			ServiceName: s.ServiceName,
			Name:        s.Name,
			Kind:        s.Kind,
			StatusCode:  s.StatusCode,
			StartTime:   s.StartTime,
			DurationMS:  s.DurationMS,
			Attributes:  s.Attributes,
		})
	}
	if err := h.core.IngestSpans(c.Request.Context(), spans); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(spans)})
}

// IngestMetricsJSON handles POST /api/v1/ingest/metrics — a simple key-authed
// JSON push for app instrumentation (no protobuf needed). Body:
//
//	{"points":[{"service":"messenger","metric":"latency_p95","value":120}]}
func (h *IngestHandler) IngestMetricsJSON(c *gin.Context) {
	var req struct {
		Points []struct {
			Service   string  `json:"service" binding:"required"`
			Metric    string  `json:"metric" binding:"required"`
			Value     float64 `json:"value"`
			Timestamp string  `json:"timestamp"`
		} `json:"points" binding:"required,dive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	raw := make([]ingest.RawMetric, 0, len(req.Points))
	for _, p := range req.Points {
		raw = append(raw, ingest.RawMetric{
			ServiceName: p.Service,
			MetricName:  p.Metric,
			Value:       p.Value,
			Timestamp:   parseTimestamp(p.Timestamp),
		})
	}
	h.writeMetrics(c, raw)
}

// IngestLogsJSON handles POST /api/v1/ingest/logs — a simple key-authed JSON
// push, used by the host agent to ship container logs. Body:
//
//	{"logs":[{"service":"gateway","level":"error","message":"…","timestamp":"…"}]}
func (h *IngestHandler) IngestLogsJSON(c *gin.Context) {
	var req struct {
		Logs []struct {
			Service   string `json:"service" binding:"required"`
			Level     string `json:"level"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
		} `json:"logs" binding:"required,dive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	projectID := middleware.IngestProjectID(c)
	cache := newServiceCache(h.core, projectID, middleware.IngestKeyID(c))
	logs := make([]entities.LogEntry, 0, len(req.Logs))
	for _, l := range req.Logs {
		if anonService(l.Service) {
			continue
		}
		sid, name := cache.resolve(c, l.Service)
		logs = append(logs, entities.LogEntry{
			ProjectID:   projectID.String(),
			ServiceID:   sid.String(),
			ServiceName: name,
			Level:       normalizeLevel(l.Level),
			Message:     l.Message,
			Metadata:    "{}",
			Timestamp:   parseTimestamp(l.Timestamp),
		})
	}
	if err := h.core.IngestLogs(c.Request.Context(), logs); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(logs)})
}

func normalizeLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error", "err", "fatal", "panic", "critical":
		return "error"
	case "warn", "warning":
		return "warning"
	default:
		return "info"
	}
}

// PromRemoteWrite handles POST /api/v1/prom/write
func (h *IngestHandler) PromRemoteWrite(c *gin.Context) {
	body, ok := readBody(c)
	if !ok {
		return
	}
	raw, err := ingest.DecodePromRemoteWrite(body)
	if err != nil {
		badRequest(c, err)
		return
	}
	h.writeMetrics(c, raw)
}

func (h *IngestHandler) writeMetrics(c *gin.Context, raw []ingest.RawMetric) {
	projectID := middleware.IngestProjectID(c)
	cache := newServiceCache(h.core, projectID, middleware.IngestKeyID(c))
	points := make([]entities.MetricPoint, 0, len(raw))
	for _, m := range raw {
		if anonService(m.ServiceName) {
			continue
		}
		sid, name := cache.resolve(c, m.ServiceName)
		points = append(points, entities.MetricPoint{
			ProjectID:   projectID.String(),
			ServiceID:   sid.String(),
			ServiceName: name,
			MetricName:  m.MetricName,
			Value:       m.Value,
			Timestamp:   m.Timestamp,
		})
	}
	if err := h.core.IngestMetrics(c.Request.Context(), points); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(points)})
}

// readBody reads the (optionally gzip-encoded) request body with a size cap.
func readBody(c *gin.Context) ([]byte, bool) {
	var reader io.Reader = io.LimitReader(c.Request.Body, maxIngestBody)
	if c.GetHeader("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			badRequest(c, err)
			return nil, false
		}
		defer gz.Close()
		reader = gz
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		badRequest(c, err)
		return nil, false
	}
	return buf.Bytes(), true
}

// serviceCache resolves service names to ids once per request. New services
// are attributed to keyID so they can be cleaned up when the key is revoked.
type serviceCache struct {
	core      *services.CoreService
	projectID uuid.UUID
	keyID     uuid.UUID
	byName    map[string]uuid.UUID
}

func newServiceCache(core *services.CoreService, projectID, keyID uuid.UUID) *serviceCache {
	return &serviceCache{core: core, projectID: projectID, keyID: keyID, byName: map[string]uuid.UUID{}}
}

func (sc *serviceCache) resolve(c *gin.Context, name string) (uuid.UUID, string) {
	if name == "" {
		name = "unknown"
	}
	if id, ok := sc.byName[name]; ok {
		return id, name
	}
	id, err := sc.core.ResolveService(c.Request.Context(), sc.projectID, name, "production", sc.keyID)
	if err != nil {
		return uuid.Nil, name
	}
	sc.byName[name] = id
	return id, name
}
