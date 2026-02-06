package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type sourceRepo struct {
	queries *sqlc.Queries
}

func NewSourceRepo(pool *pgxpool.Pool) port.SourceRepo {
	return &sourceRepo{
		queries: sqlc.New(pool),
	}
}

func (r *sourceRepo) Create(ctx context.Context, params sqlc.CreateSourceParams) (*domain.Source, error) {
	source, err := r.queries.CreateSource(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.domainSource(source), nil
}

func (r *sourceRepo) GetByID(ctx context.Context, id string) (*domain.Source, error) {
	source, err := r.queries.GetSourceByID(ctx, pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true})
	if err != nil {
		return nil, err
	}

	return r.domainSource(source), nil
}

func (r *sourceRepo) domainSource(s sqlc.Source) *domain.Source {
	var metadata map[string]interface{}
	json.Unmarshal(s.Metadata, &metadata)

	return &domain.Source{
		ID:       uuid.UUID(s.ID.Bytes).String(),
		Type:     domain.SourceType(s.Type),
		Metadata: metadata,
	}
}
