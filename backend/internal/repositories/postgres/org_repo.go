package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgRepo struct{ db *pgxpool.Pool }

func NewOrgRepo(db *pgxpool.Pool) *OrgRepo { return &OrgRepo{db: db} }

func (r *OrgRepo) Create(ctx context.Context, org *entities.Organization) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO organizations (name, slug, created_by)
		 VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		org.Name, org.Slug, org.CreatedBy,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
}

func (r *OrgRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]entities.Organization, error) {
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.created_by, o.created_at, o.updated_at, tm.role
		 FROM organizations o
		 JOIN team_members tm ON tm.organization_id = o.id
		 WHERE tm.user_id = $1
		 ORDER BY o.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Organization
	for rows.Next() {
		var o entities.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt, &o.Role); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error) {
	o := &entities.Organization{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, created_by, created_at, updated_at FROM organizations WHERE id=$1`,
		id,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (r *OrgRepo) AddMember(ctx context.Context, m *entities.TeamMember) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO team_members (organization_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role
		 RETURNING id, created_at`,
		m.OrganizationID, m.UserID, m.Role,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *OrgRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]entities.TeamMember, error) {
	rows, err := r.db.Query(ctx,
		`SELECT tm.id, tm.organization_id, tm.user_id, u.email, u.name, tm.role, tm.created_at
		 FROM team_members tm JOIN users u ON u.id = tm.user_id
		 WHERE tm.organization_id = $1 ORDER BY tm.created_at`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.TeamMember
	for rows.Next() {
		var m entities.TeamMember
		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.Email, &m.Name, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *OrgRepo) GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (entities.OrgRole, error) {
	var role entities.OrgRole
	err := r.db.QueryRow(ctx,
		`SELECT role FROM team_members WHERE organization_id=$1 AND user_id=$2`,
		orgID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}
