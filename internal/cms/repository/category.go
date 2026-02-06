package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type categoryRepo struct {
	queries *sqlc.Queries
}

func NewCategoryRepo(pool *pgxpool.Pool) port.CategoryRepo {
	return &categoryRepo{
		queries: sqlc.New(pool),
	}
}

func (r *categoryRepo) Create(ctx context.Context, params sqlc.CreateCategoryParams) (*domain.Category, error) {
	category, err := r.queries.CreateCategory(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.domainCategory(category), nil
}

func (r *categoryRepo) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	category, err := r.queries.GetCategoryByID(ctx, pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true})
	if err != nil {
		return nil, err
	}

	return r.domainCategory(category), nil
}

func (r *categoryRepo) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	category, err := r.queries.GetCategoryByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return r.domainCategory(category), nil
}

func (r *categoryRepo) List(ctx context.Context) ([]*domain.Category, error) {
	categories, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Category, len(categories))
	for i, c := range categories {
		result[i] = r.domainCategory(c)
	}

	return result, nil
}

func (r *categoryRepo) domainCategory(c sqlc.Category) *domain.Category {
	return &domain.Category{
		ID:          uuid.UUID(c.ID.Bytes).String(),
		Name:        c.Name,
		Description: c.Description.String,
	}
}
