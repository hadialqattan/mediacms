package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

func (s *Service) ImportProgram(ctx context.Context, sourceType domain.SourceType, metadata map[string]interface{}) (*domain.Program, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	source, err := s.sourceRepo.WithTx(tx).Create(ctx, sqlc.CreateSourceParams{
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

	program, err := s.programRepo.WithTx(tx).Create(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return program, nil
}
