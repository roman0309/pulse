package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/ingest"
	"github.com/acme/observability/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
	cache := newServiceCache(h.core, projectID)
	logs := make([]entities.LogEntry, 0, len(raw))
	for _, l := range raw {
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

// OTLPTraces handles POST /otlp/v1/traces — accepted but not yet stored (Phase 3).
func (h *IngestHandler) OTLPTraces(c *gin.Context) {
	_, _ = io.Copy(io.Discard, c.Request.Body)
	c.JSON(http.StatusAccepted, gin.H{"status": "traces accepted (storage pending)"})
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
	cache := newServiceCache(h.core, projectID)
	points := make([]entities.MetricPoint, 0, len(raw))
	for _, m := range raw {
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

// serviceCache resolves service names to ids once per request.
type serviceCache struct {
	core      *services.CoreService
	projectID uuid.UUID
	byName    map[string]uuid.UUID
}

func newServiceCache(core *services.CoreService, projectID uuid.UUID) *serviceCache {
	return &serviceCache{core: core, projectID: projectID, byName: map[string]uuid.UUID{}}
}

func (sc *serviceCache) resolve(c *gin.Context, name string) (uuid.UUID, string) {
	if name == "" {
		name = "unknown"
	}
	if id, ok := sc.byName[name]; ok {
		return id, name
	}
	id, err := sc.core.ResolveService(c.Request.Context(), sc.projectID, name, "production")
	if err != nil {
		return uuid.Nil, name
	}
	sc.byName[name] = id
	return id, name
}
