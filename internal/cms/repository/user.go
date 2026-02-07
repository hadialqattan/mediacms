package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type userRepo struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserRepo(pool *pgxpool.Pool) port.UserRepo {
	return &userRepo{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *userRepo) WithTx(tx pgx.Tx) port.UserRepo {
	return &userRepo{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *userRepo) Create(ctx context.Context, params sqlc.CreateUserParams) (*domain.User, error) {
	user, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.domainUser(user), nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := r.queries.GetUserByID(ctx, pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true})
	if err != nil {
		return nil, err
	}

	return r.domainUser(user), nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return r.domainUser(user), nil
}

func (r *userRepo) domainUser(u sqlc.User) *domain.User {
	return &domain.User{
		ID:           uuid.UUID(u.ID.Bytes).String(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         domain.UserRole(u.Role),
		CreatedAt:    u.CreatedAt.Time,
	}
}
