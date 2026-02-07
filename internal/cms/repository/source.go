package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/port"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type sourceRepo struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewSourceRepo(db sqlc.DBTX) port.SourceRepo {
	return &sourceRepo{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *sourceRepo) WithTx(tx pgx.Tx) port.SourceRepo {
	return &sourceRepo{
		db:      r.db,
		queries: r.queries.WithTx(tx),
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
