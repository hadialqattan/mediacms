package service

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
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
