package analyzer

import (
	"context"
	"time"

	"github.com/acme/observability/internal/domain/entities"
)

// Input is the aggregated signal set fed to a root-cause analyzer.
type Input struct {
	ProjectID   string
	Window      time.Duration
	Now         time.Time
	Metrics     []entities.MetricSeries
	Logs        []entities.LogEntry
	Alerts      []entities.Alert
	Deployments []entities.Deployment
	Services    []entities.Service
}

// Result is a natural-language root-cause summary with supporting evidence.
type Result struct {
	Summary    string   `json:"summary"`
	Confidence float64  `json:"confidence"` // 0..1
	Evidence   []string `json:"evidence"`
	Findings   []string `json:"findings"`
}

// Analyzer is the abstraction that lets us swap the deterministic engine for an
// LLM-backed one later without touching callers.
type Analyzer interface {
	Analyze(ctx context.Context, in Input) (*Result, error)
}
