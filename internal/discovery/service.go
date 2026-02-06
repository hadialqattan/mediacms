package discovery

import (
	"context"

	"thmanyah.com/content-platform/internal/discovery/port"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type Service struct {
	searchIndex port.SearchIndex
}

func NewService(searchIndex port.SearchIndex) *Service {
	return &Service{
		searchIndex: searchIndex,
	}
}

func (s *Service) SearchPrograms(ctx context.Context, params port.SearchParams) (*port.SearchResult, error) {
	return s.searchIndex.SearchPrograms(ctx, params)
}

func (s *Service) GetProgram(ctx context.Context, id string) (*domain.Program, error) {
	return s.searchIndex.GetProgram(ctx, id)
}
