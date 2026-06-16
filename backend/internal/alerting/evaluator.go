// Package alerting contains the background engine that evaluates alert rules
// against live metrics, fires/resolves alerts, and dispatches notifications.
package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/repositories"
	"github.com/acme/observability/internal/ws"
	"github.com/acme/observability/pkg/notify"
	"github.com/google/uuid"
)

// Evaluator periodically checks every enabled rule and manages alert lifecycle.
type Evaluator struct {
	Rules    repositories.AlertRuleRepository
	Metrics  repositories.MetricRepository
	Alerts   repositories.AlertRepository
	Timeline repositories.TimelineRepository
	Services repositories.ServiceRepository
	Hub      *ws.Hub
	Notifier *notify.Notifier
	Interval time.Duration
	Window   time.Duration
	Log      *slog.Logger
}

// Run blocks, evaluating rules every Interval until ctx is cancelled.
func (e *Evaluator) Run(ctx context.Context) {
	if e.Interval <= 0 {
		e.Interval = 15 * time.Second
	}
	if e.Window <= 0 {
		e.Window = 2 * time.Minute
	}
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()
	e.Log.Info("alert evaluator started", "interval", e.Interval.String())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.evaluateAll(ctx)
		}
	}
}

func (e *Evaluator) evaluateAll(ctx context.Context) {
	rules, err := e.Rules.ListEnabled(ctx)
	if err != nil {
		e.Log.Error("list enabled rules", "err", err)
		return
	}
	for i := range rules {
		if err := e.evaluateRule(ctx, &rules[i]); err != nil {
			e.Log.Error("evaluate rule", "rule", rules[i].Name, "err", err)
		}
	}
}

func (e *Evaluator) evaluateRule(ctx context.Context, rule *entities.AlertRule) error {
	serviceID := ""
	if rule.ServiceID != nil {
		serviceID = rule.ServiceID.String()
	}
	value, ok, err := e.Metrics.EvalValue(ctx, rule.ProjectID.String(), serviceID, rule.Metric, time.Now().Add(-e.Window))
	if err != nil {
		return err
	}
	if !ok {
		return nil // no data — leave state untouched
	}

	now := time.Now()
	breached := rule.Breached(value)

	if !breached {
		// Recover if currently firing.
		if rule.ActiveAlertID != nil {
			if err := e.resolve(ctx, rule, value); err != nil {
				return err
			}
		}
		if rule.BreachingSince != nil || rule.ActiveAlertID != nil {
			return e.Rules.SetState(ctx, rule.ID, nil, nil)
		}
		return nil
	}

	// Breaching.
	if rule.BreachingSince == nil {
		rule.BreachingSince = &now
		if err := e.Rules.SetState(ctx, rule.ID, &now, rule.ActiveAlertID); err != nil {
			return err
		}
	}
	if rule.ActiveAlertID != nil {
		return nil // already firing
	}
	// Honor the "for" duration before firing.
	if now.Sub(*rule.BreachingSince) < time.Duration(rule.ForSeconds)*time.Second {
		return nil
	}
	return e.fire(ctx, rule, value)
}

func (e *Evaluator) fire(ctx context.Context, rule *entities.AlertRule, value float64) error {
	svcName := e.serviceName(ctx, rule.ServiceID)
	desc := fmt.Sprintf("%s is %.1f (rule: %s %s %.1f)", rule.Metric, value, rule.Metric, rule.Operator, rule.Threshold)

	alert := &entities.Alert{
		ProjectID:   rule.ProjectID,
		ServiceID:   rule.ServiceID,
		Title:       rule.Name,
		Type:        rule.Type,
		Severity:    rule.Severity,
		Status:      entities.AlertActive,
		Description: desc,
		CreatedAt:   time.Now(),
	}
	if err := e.Alerts.Create(ctx, alert); err != nil {
		return err
	}

	sev := rule.Severity
	ev := &entities.TimelineEvent{
		ProjectID:   rule.ProjectID,
		ServiceID:   rule.ServiceID,
		Type:        entities.EventAlert,
		Title:       "Alert triggered",
		Description: rule.Name + " — " + desc,
		Severity:    &sev,
		RefID:       &alert.ID,
		OccurredAt:  alert.CreatedAt,
	}
	_ = e.Timeline.Create(ctx, ev)

	e.broadcast(rule.ProjectID, "alert", alert)
	e.broadcast(rule.ProjectID, "timeline", ev)
	e.notify(ctx, rule, svcName, value, "firing", desc)

	e.Log.Info("alert fired", "rule", rule.Name, "value", value)
	return e.Rules.SetState(ctx, rule.ID, rule.BreachingSince, &alert.ID)
}

func (e *Evaluator) resolve(ctx context.Context, rule *entities.AlertRule, value float64) error {
	now := time.Now()
	if err := e.Alerts.Resolve(ctx, *rule.ActiveAlertID, now); err != nil {
		return err
	}
	svcName := e.serviceName(ctx, rule.ServiceID)
	sev := rule.Severity
	ev := &entities.TimelineEvent{
		ProjectID:   rule.ProjectID,
		ServiceID:   rule.ServiceID,
		Type:        entities.EventRecovery,
		Title:       "Service recovered",
		Description: fmt.Sprintf("%s back to normal (%.1f)", rule.Metric, value),
		Severity:    &sev,
		OccurredAt:  now,
	}
	_ = e.Timeline.Create(ctx, ev)
	e.broadcast(rule.ProjectID, "timeline", ev)
	e.notify(ctx, rule, svcName, value, "resolved", ev.Description)
	e.Log.Info("alert resolved", "rule", rule.Name, "value", value)
	return nil
}

func (e *Evaluator) notify(ctx context.Context, rule *entities.AlertRule, service string, value float64, status, desc string) {
	if e.Notifier == nil || rule.NotifyType == "" || rule.NotifyType == "none" {
		return
	}
	err := e.Notifier.Send(ctx, rule.NotifyType, rule.NotifyURL, notify.Message{
		Title:       rule.Name,
		Service:     service,
		Metric:      rule.Metric,
		Value:       value,
		Threshold:   rule.Threshold,
		Severity:    string(rule.Severity),
		Status:      status,
		Description: desc,
	})
	if err != nil {
		e.Log.Warn("notification failed", "rule", rule.Name, "err", err)
	}
}

func (e *Evaluator) serviceName(ctx context.Context, id *uuid.UUID) string {
	if id == nil {
		return "all services"
	}
	if svc, err := e.Services.GetByID(ctx, *id); err == nil {
		return svc.Name
	}
	return ""
}

func (e *Evaluator) broadcast(projectID uuid.UUID, kind string, payload interface{}) {
	if e.Hub != nil {
		e.Hub.Broadcast(ws.Event{Type: kind, ProjectID: projectID.String(), Payload: payload})
	}
}
