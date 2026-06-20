package postgres

import (
	"context"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRuleRepo struct{ db *pgxpool.Pool }

func NewAlertRuleRepo(db *pgxpool.Pool) *AlertRuleRepo { return &AlertRuleRepo{db: db} }

const ruleCols = `id, project_id, name, service_id, metric, operator, threshold, for_seconds,
	severity, type, notify_type, notify_url, notify_channel_id, enabled, breaching_since, active_alert_id,
	created_at, updated_at`

func scanRule(row pgx.Row) (*entities.AlertRule, error) {
	r := &entities.AlertRule{}
	err := row.Scan(&r.ID, &r.ProjectID, &r.Name, &r.ServiceID, &r.Metric, &r.Operator,
		&r.Threshold, &r.ForSeconds, &r.Severity, &r.Type, &r.NotifyType, &r.NotifyURL,
		&r.NotifyChannelID, &r.Enabled, &r.BreachingSince, &r.ActiveAlertID, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *AlertRuleRepo) Create(ctx context.Context, a *entities.AlertRule) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO alert_rules
		   (project_id, name, service_id, metric, operator, threshold, for_seconds, severity, type, notify_type, notify_url, notify_channel_id, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id, created_at, updated_at`,
		a.ProjectID, a.Name, a.ServiceID, a.Metric, a.Operator, a.Threshold, a.ForSeconds,
		a.Severity, a.Type, a.NotifyType, a.NotifyURL, a.NotifyChannelID, a.Enabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AlertRuleRepo) Update(ctx context.Context, a *entities.AlertRule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_rules SET name=$3, service_id=$4, metric=$5, operator=$6, threshold=$7,
		   for_seconds=$8, severity=$9, type=$10, notify_type=$11, notify_url=$12, notify_channel_id=$13, enabled=$14
		 WHERE id=$1 AND project_id=$2`,
		a.ID, a.ProjectID, a.Name, a.ServiceID, a.Metric, a.Operator, a.Threshold,
		a.ForSeconds, a.Severity, a.Type, a.NotifyType, a.NotifyURL, a.NotifyChannelID, a.Enabled,
	)
	return err
}

func (r *AlertRuleRepo) Delete(ctx context.Context, projectID, ruleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM alert_rules WHERE id=$1 AND project_id=$2`, ruleID, projectID)
	return err
}

func (r *AlertRuleRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.AlertRule, error) {
	rows, err := r.db.Query(ctx, `SELECT `+ruleCols+` FROM alert_rules WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRules(rows)
}

func (r *AlertRuleRepo) ListEnabled(ctx context.Context) ([]entities.AlertRule, error) {
	rows, err := r.db.Query(ctx, `SELECT `+ruleCols+` FROM alert_rules WHERE enabled=true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRules(rows)
}

func (r *AlertRuleRepo) SetState(ctx context.Context, ruleID uuid.UUID, breachingSince *time.Time, activeAlertID *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alert_rules SET breaching_since=$2, active_alert_id=$3 WHERE id=$1`,
		ruleID, breachingSince, activeAlertID,
	)
	return err
}

func collectRules(rows pgx.Rows) ([]entities.AlertRule, error) {
	var out []entities.AlertRule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
