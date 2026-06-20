package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelRepo struct{ db *pgxpool.Pool }

func NewChannelRepo(db *pgxpool.Pool) *ChannelRepo { return &ChannelRepo{db: db} }

const channelCols = `id, project_id, name, type, config_enc, created_at`

func scanChannel(row pgx.Row) (*entities.NotificationChannel, error) {
	ch := &entities.NotificationChannel{}
	err := row.Scan(&ch.ID, &ch.ProjectID, &ch.Name, &ch.Type, &ch.ConfigEnc, &ch.CreatedAt)
	return ch, err
}

func (r *ChannelRepo) Create(ctx context.Context, ch *entities.NotificationChannel) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO notification_channels (project_id, name, type, config_enc)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at`,
		ch.ProjectID, ch.Name, ch.Type, ch.ConfigEnc,
	).Scan(&ch.ID, &ch.CreatedAt)
}

func (r *ChannelRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.NotificationChannel, error) {
	rows, err := r.db.Query(ctx, `SELECT `+channelCols+` FROM notification_channels WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entities.NotificationChannel
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ch)
	}
	return out, rows.Err()
}

func (r *ChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.NotificationChannel, error) {
	ch, err := scanChannel(r.db.QueryRow(ctx, `SELECT `+channelCols+` FROM notification_channels WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ch, err
}

func (r *ChannelRepo) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notification_channels WHERE id=$1 AND project_id=$2`, id, projectID)
	return err
}
