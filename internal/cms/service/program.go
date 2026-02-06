package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

func (s *Service) CreateProgram(ctx context.Context, params sqlc.CreateProgramParams) (*domain.Program, error) {
	return s.programRepo.Create(ctx, params)
}

func (s *Service) GetProgram(ctx context.Context, id string) (*domain.Program, error) {
	return s.programRepo.GetByID(ctx, id)
}

func (s *Service) GetProgramBySlug(ctx context.Context, slug string) (*domain.Program, error) {
	return s.programRepo.GetBySlug(ctx, slug)
}

func (s *Service) ListPrograms(ctx context.Context) ([]*domain.Program, error) {
	return s.programRepo.List(ctx)
}

func (s *Service) UpdateProgram(ctx context.Context, id string, params sqlc.UpdateProgramParams) (*domain.Program, error) {
	return s.programRepo.Update(ctx, id, params)
}

func (s *Service) UpdateProgramBySlug(ctx context.Context, slug string, params sqlc.UpdateProgramParams) (*domain.Program, error) {
	program, err := s.programRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return s.programRepo.Update(ctx, program.ID, params)
}

func (s *Service) PublishProgram(ctx context.Context, id, userID string) (*domain.Program, error) {
	program, err := s.programRepo.Publish(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if err := s.emitOutboxEvent(ctx, domain.OutboxEventTypeProgramUpsert, &program.ID); err != nil {
		return nil, err
	}

	return program, nil
}

func (s *Service) PublishProgramBySlug(ctx context.Context, slug, userID string) (*domain.Program, error) {
	program, err := s.programRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return s.PublishProgram(ctx, program.ID, userID)
}

func (s *Service) DeleteProgram(ctx context.Context, id, userID string) error {
	program, err := s.programRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.programRepo.Delete(ctx, id, userID); err != nil {
		return err
	}

	if program.IsPublished() {
		if err := s.emitOutboxEvent(ctx, domain.OutboxEventTypeProgramDelete, &program.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) DeleteProgramBySlug(ctx context.Context, slug, userID string) error {
	program, err := s.programRepo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	return s.DeleteProgram(ctx, program.ID, userID)
}

func (s *Service) AssignCategories(ctx context.Context, programID string, categoryIDs []string) error {
	if err := s.programRepo.AssignCategories(ctx, programID, categoryIDs); err != nil {
		return err
	}

	program, err := s.programRepo.GetByID(ctx, programID)
	if err != nil {
		return err
	}

	if program.IsPublished() {
		if err := s.emitOutboxEvent(ctx, domain.OutboxEventTypeProgramUpsert, &programID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) AssignCategoriesBySlug(ctx context.Context, slug string, categoryIDs []string) error {
	program, err := s.programRepo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	return s.AssignCategories(ctx, program.ID, categoryIDs)
}

func (s *Service) GetProgramCategories(ctx context.Context, programID string) ([]domain.Category, error) {
	return s.programRepo.GetCategories(ctx, programID)
}

func (s *Service) GetProgramCategoriesBySlug(ctx context.Context, slug string) ([]domain.Category, error) {
	program, err := s.programRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return s.programRepo.GetCategories(ctx, program.ID)
}

func (s *Service) emitOutboxEvent(ctx context.Context, eventType domain.OutboxEventType, programID *string) error {
	payload, err := json.Marshal(map[string]interface{}{"program_id": *programID})
	if err != nil {
		return err
	}

	_, err = s.outboxRepo.Create(ctx, sqlc.CreateOutboxEventParams{
		Type:      string(eventType),
		Payload:   payload,
		ProgramID: pgtype.UUID{Bytes: uuid.MustParse(*programID), Valid: true},
	})
	return err
}
