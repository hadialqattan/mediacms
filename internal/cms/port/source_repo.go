package port

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type SourceRepo interface {
	Create(ctx context.Context, params sqlc.CreateSourceParams) (*domain.Source, error)
	GetByID(ctx context.Context, id string) (*domain.Source, error)
	WithTx(tx pgx.Tx) SourceRepo
}
