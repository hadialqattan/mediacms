package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

func (s *Service) ImportProgram(ctx context.Context, sourceType domain.SourceType, metadata map[string]interface{}) (*domain.Program, error) {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	source, err := s.sourceRepo.Create(ctx, sqlc.CreateSourceParams{
		Type:     string(sourceType),
		Metadata: metadataBytes,
	})
	if err != nil {
		return nil, err
	}

	createdByID := uuid.MustParse(metadata["created_by"].(string))
	params := sqlc.CreateProgramParams{
		Slug:        uuid.New().String(),
		Title:       metadata["title"].(string),
		Description: pgtype.Text{String: "", Valid: false},
		Type:        string(domain.ProgramTypePodcast),
		Language:    string(domain.LanguageEn),
		DurationMs:  0,
		SourceID:    pgtype.UUID{Bytes: uuid.MustParse(source.ID), Valid: true},
		CreatedBy:   pgtype.UUID{Bytes: createdByID, Valid: true},
	}

	return s.programRepo.Create(ctx, params)
}
