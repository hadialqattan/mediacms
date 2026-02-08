package discovery

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/discovery/port"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
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

func (s *Service) GetRecentPrograms(ctx context.Context, params port.RecentParams) (*port.RecentResult, error) {
	return s.searchIndex.GetRecentPrograms(ctx, params)
}

func (s *Service) GetFacets(ctx context.Context) (*port.FacetsResult, error) {
	return s.searchIndex.GetFacets(ctx)
}
