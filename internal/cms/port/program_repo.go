package port

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type ProgramRepo interface {
	WithTx(tx pgx.Tx) ProgramRepo
	Create(ctx context.Context, params sqlc.CreateProgramParams) (*domain.Program, error)
	GetByID(ctx context.Context, id string) (*domain.Program, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Program, error)
	Update(ctx context.Context, id string, params sqlc.UpdateProgramParams) (*domain.Program, error)
	Publish(ctx context.Context, id, publishedBy string) (*domain.Program, error)
	Delete(ctx context.Context, id, deletedBy string) error
}
