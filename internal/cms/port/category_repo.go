package port

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type CategoryRepo interface {
	Create(ctx context.Context, params sqlc.CreateCategoryParams) (*domain.Category, error)
	GetByID(ctx context.Context, id string) (*domain.Category, error)
	GetByName(ctx context.Context, name string) (*domain.Category, error)
	List(ctx context.Context) ([]*domain.Category, error)
	WithTx(tx pgx.Tx) CategoryRepo
}
