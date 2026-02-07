package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	tx, err := s.transactionPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	program, err := s.programRepo.WithTx(tx).Publish(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if err := s.emitOutboxEvent(ctx, tx, domain.OutboxEventTypeProgramUpsert, program); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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

	tx, err := s.transactionPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.programRepo.WithTx(tx).Delete(ctx, id, userID); err != nil {
		return err
	}
	if program.IsPublished() {
		if err := s.emitOutboxEvent(ctx, tx, domain.OutboxEventTypeProgramDelete, program); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
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
	tx, err := s.transactionPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.programRepo.WithTx(tx).AssignCategories(ctx, programID, categoryIDs); err != nil {
		return err
	}
	program, err := s.programRepo.WithTx(tx).GetByID(ctx, programID)
	if err != nil {
		return err
	}
	if program.IsPublished() {
		if err := s.emitOutboxEvent(ctx, tx, domain.OutboxEventTypeProgramUpsert, program); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
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

func (s *Service) emitOutboxEvent(ctx context.Context, tx pgx.Tx, eventType domain.OutboxEventType, program *domain.Program) error {
	categories, err := s.programRepo.WithTx(tx).GetCategories(ctx, program.ID)
	if err != nil {
		return err
	}
	program.Categories = categories

	payload, err := json.Marshal(program)
	if err != nil {
		return err
	}

	_, err = s.outboxRepo.WithTx(tx).Create(ctx, sqlc.CreateOutboxEventParams{
		Type:      string(eventType),
		Payload:   payload,
		ProgramID: pgtype.UUID{Bytes: uuid.MustParse(program.ID), Valid: true},
	})
	return err
}
