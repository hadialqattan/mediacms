package service

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

func (s *Service) CreateCategory(ctx context.Context, params sqlc.CreateCategoryParams) (*domain.Category, error) {
	return s.categoryRepo.Create(ctx, params)
}

func (s *Service) GetCategory(ctx context.Context, id string) (*domain.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *Service) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	return s.categoryRepo.List(ctx)
}
