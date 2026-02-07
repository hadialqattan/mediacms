package port

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type OutboxRepo interface {
	Create(ctx context.Context, params sqlc.CreateOutboxEventParams) (*domain.OutboxEvent, error)
	WithTx(tx pgx.Tx) OutboxRepo
}
