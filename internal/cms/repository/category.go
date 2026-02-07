package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/port"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type categoryRepo struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewCategoryRepo(db sqlc.DBTX) port.CategoryRepo {
	return &categoryRepo{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *categoryRepo) WithTx(tx pgx.Tx) port.CategoryRepo {
	return &categoryRepo{
		db:      r.db,
		queries: r.queries.WithTx(tx),
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
