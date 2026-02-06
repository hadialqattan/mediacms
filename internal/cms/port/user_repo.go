package port

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type UserRepo interface {
	Create(ctx context.Context, params sqlc.CreateUserParams) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}
