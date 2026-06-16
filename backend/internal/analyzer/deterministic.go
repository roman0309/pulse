package analyzer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/acme/observability/internal/domain/entities"
)

// DeterministicAnalyzer correlates deployments, metric spikes, error logs and
// alerts using simple heuristics. It produces a human-readable RCA summary
// without any LLM. It implements the Analyzer interface.
type DeterministicAnalyzer struct{}

func NewDeterministic() *DeterministicAnalyzer { return &DeterministicAnalyzer{} }

func (a *DeterministicAnalyzer) Analyze(_ context.Context, in Input) (*Result, error) {
	var (
		evidence   []string
		findings   []string
		confidence float64
	)

	// 1. Detect latency regression from the p95 series (compare recent vs baseline).
	latencyDelta, latencyService := detectLatencyRegression(in.Metrics)
	if latencyDelta > 0 {
		findings = append(findings, fmt.Sprintf("%s latency increased %.0f%%", displayName(latencyService, in.Services), latencyDelta))
		evidence = append(evidence, fmt.Sprintf("P95 latency rose by %.0f%% over the analysis window", latencyDelta))
		confidence += 0.3
	}

	// 2. Detect error-rate increase.
	errorDelta, _ := detectErrorRegression(in.Metrics)
	if errorDelta > 0 {
		findings = append(findings, fmt.Sprintf("error rate increased %.0f%%", errorDelta))
		confidence += 0.2
	}

	// 3. Correlate with the most recent deployment preceding the regression.
	recentDeploy := mostRecentDeployment(in.Deployments, in.Now)
	if recentDeploy != nil {
		evidence = append(evidence, fmt.Sprintf("Deployment %s was released %s before the regression",
			recentDeploy.Version, humanizeSince(recentDeploy.CreatedAt, in.Now)))
		confidence += 0.25
	}

	// 4. Inspect error logs for dominant failure mode.
	topPattern, count := dominantErrorPattern(in.Logs)
	if topPattern != "" {
		findings = append(findings, fmt.Sprintf("most failures originated from %s", topPattern))
		evidence = append(evidence, fmt.Sprintf("%d error logs matched \"%s\"", count, topPattern))
		confidence += 0.15
	}

	// 5. Active alerts reinforce the signal.
	activeAlerts := 0
	for _, al := range in.Alerts {
		if al.Status == entities.AlertActive {
			activeAlerts++
		}
	}
	if activeAlerts > 0 {
		evidence = append(evidence, fmt.Sprintf("%d active alert(s) currently firing", activeAlerts))
		confidence += 0.1
	}

	if confidence > 1 {
		confidence = 1
	}

	summary := buildSummary(latencyService, in.Services, latencyDelta, recentDeploy, topPattern)
	if len(findings) == 0 {
		summary = "No significant anomalies detected in the selected window. The system appears to be operating within normal parameters."
		confidence = 0.6
	}

	return &Result{
		Summary:    summary,
		Confidence: confidence,
		Evidence:   evidence,
		Findings:   findings,
	}, nil
}

func buildSummary(serviceID string, services []entities.Service, latencyDelta float64, dep *entities.Deployment, pattern string) string {
	var b strings.Builder
	name := displayName(serviceID, services)
	if latencyDelta > 0 {
		b.WriteString(fmt.Sprintf("%s latency increased %.0f%%", name, latencyDelta))
		if dep != nil {
			b.WriteString(fmt.Sprintf(" after deployment %s.", dep.Version))
		} else {
			b.WriteString(".")
		}
	} else if dep != nil {
		b.WriteString(fmt.Sprintf("Anomalies detected following deployment %s.", dep.Version))
	} else {
		b.WriteString("Anomalies detected in the selected window.")
	}
	if pattern != "" {
		b.WriteString(fmt.Sprintf(" Most failures originated from %s.", pattern))
	}
	return b.String()
}

// ---------- heuristics ----------

func detectLatencyRegression(series []entities.MetricSeries) (float64, string) {
	return detectRegression(series, "latency_p95")
}

func detectErrorRegression(series []entities.MetricSeries) (float64, string) {
	return detectRegression(series, "error_rate")
}

// detectRegression compares the average of the most recent third of points to
// the earliest third and returns the percentage increase, if positive.
func detectRegression(series []entities.MetricSeries, metric string) (float64, string) {
	var best float64
	var bestService string
	for _, s := range series {
		if s.MetricName != metric || len(s.Points) < 6 {
			continue
		}
		n := len(s.Points)
		third := n / 3
		baseline := avg(s.Points[:third])
		recent := avg(s.Points[n-third:])
		if baseline <= 0.0001 {
			baseline = 0.0001
		}
		delta := (recent - baseline) / baseline * 100
		if delta > best {
			best = delta
			bestService = s.ServiceID
		}
	}
	return best, bestService
}

func avg(pts []entities.SeriesPoint) float64 {
	if len(pts) == 0 {
		return 0
	}
	var sum float64
	for _, p := range pts {
		sum += p.Value
	}
	return sum / float64(len(pts))
}

func mostRecentDeployment(deps []entities.Deployment, now time.Time) *entities.Deployment {
	var latest *entities.Deployment
	for i := range deps {
		d := &deps[i]
		if d.CreatedAt.After(now) {
			continue
		}
		if latest == nil || d.CreatedAt.After(latest.CreatedAt) {
			latest = d
		}
	}
	return latest
}

func dominantErrorPattern(logs []entities.LogEntry) (string, int) {
	patterns := map[string]int{
		"database timeout exceptions": 0,
		"connection refused errors":   0,
		"null pointer exceptions":     0,
		"out of memory errors":        0,
	}
	keywords := map[string]string{
		"timeout":           "database timeout exceptions",
		"connection refused": "connection refused errors",
		"null pointer":      "null pointer exceptions",
		"out of memory":     "out of memory errors",
		"oom":               "out of memory errors",
	}
	for _, l := range logs {
		if l.Level != "error" {
			continue
		}
		msg := strings.ToLower(l.Message)
		for kw, label := range keywords {
			if strings.Contains(msg, kw) {
				patterns[label]++
			}
		}
	}
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range patterns {
		if v > 0 {
			sorted = append(sorted, kv{k, v})
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
	if len(sorted) == 0 {
		return "", 0
	}
	return sorted[0].k, sorted[0].v
}

func displayName(serviceID string, services []entities.Service) string {
	for _, s := range services {
		if s.ID.String() == serviceID {
			return s.Name
		}
	}
	if serviceID == "" {
		return "The service"
	}
	return "Service"
}

func humanizeSince(t, now time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		return "moments"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f hours", d.Hours())
}
