package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

func (s *Service) CreateProgram(ctx context.Context, params sqlc.CreateProgramParams) (*domain.Program, error) {
	return s.programRepo.Create(ctx, params)
}

func (s *Service) GetProgram(ctx context.Context, id string) (*domain.Program, error) {
	return s.programRepo.GetByID(ctx, id)
}

func (s *Service) ListPrograms(ctx context.Context, limit, offset int) ([]*domain.Program, error) {
	programs, err := s.programRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	return programs, nil
}

func (s *Service) UpdateProgram(ctx context.Context, id string, params sqlc.UpdateProgramParams) (*domain.Program, error) {
	return s.programRepo.Update(ctx, id, params)
}

func (s *Service) PublishProgram(ctx context.Context, id, userID string) (*domain.Program, error) {
	tx, err := s.pool.Begin(ctx)
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

func (s *Service) DeleteProgram(ctx context.Context, id, userID string) error {
	program, err := s.programRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
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

type BulkCreateProgramFailure struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

func (s *Service) BulkCreatePrograms(ctx context.Context, programs []sqlc.CreateProgramParams, createdBy string) ([]*domain.Program, []BulkCreateProgramFailure) {
	var created []*domain.Program
	var failures []BulkCreateProgramFailure

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		for i := range programs {
			failures = append(failures, BulkCreateProgramFailure{
				Index: i,
				Error: "failed to begin transaction",
			})
		}
		return nil, failures
	}
	defer tx.Rollback(ctx)

	for i, prog := range programs {
		if !domain.IsValidProgramType(prog.Type) {
			failures = append(failures, BulkCreateProgramFailure{
				Index: i,
				Error: "invalid type: must be 'podcast' or 'documentary'",
			})
			continue
		}

		if !domain.IsValidProgramLanguage(prog.Language) {
			failures = append(failures, BulkCreateProgramFailure{
				Index: i,
				Error: "invalid language: must be 'ar' or 'en'",
			})
			continue
		}

		tags := prog.Tags
		if tags == nil {
			tags = []string{}
		}

		program, err := s.programRepo.WithTx(tx).Create(ctx, sqlc.CreateProgramParams{
			Slug:        prog.Slug,
			Title:       prog.Title,
			Description: prog.Description,
			Type:        prog.Type,
			Language:    prog.Language,
			DurationMs:  int32(prog.DurationMs),
			Tags:        tags,
			CreatedBy:   pgtype.UUID{Bytes: uuid.MustParse(createdBy), Valid: true},
		})
		if err != nil {
			failures = append(failures, BulkCreateProgramFailure{
				Index: i,
				Error: fmt.Sprintf("failed to create: %v", err),
			})
			continue
		}

		created = append(created, program)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, []BulkCreateProgramFailure{{Error: "failed to commit transaction"}}
	}

	return created, failures
}

type BulkDeleteProgramFailure struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

func (s *Service) BulkDeletePrograms(ctx context.Context, ids []string, deletedBy string) ([]string, []BulkDeleteProgramFailure) {
	var deleted []string
	var failures []BulkDeleteProgramFailure

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		for _, id := range ids {
			failures = append(failures, BulkDeleteProgramFailure{
				ID:    id,
				Error: "failed to begin transaction",
			})
		}
		return nil, failures
	}
	defer tx.Rollback(ctx)

	for _, id := range ids {
		savepointName := fmt.Sprintf("sp_%s", id)
		if _, err := tx.Exec(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName)); err != nil {
			failures = append(failures, BulkDeleteProgramFailure{
				ID:    id,
				Error: fmt.Sprintf("failed to create savepoint: %v", err),
			})
			continue
		}

		program, err := s.programRepo.WithTx(tx).GetByID(ctx, id)
		if err != nil {
			tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
			failures = append(failures, BulkDeleteProgramFailure{
				ID:    id,
				Error: "program not found",
			})
			continue
		}

		if err := s.programRepo.WithTx(tx).Delete(ctx, id, deletedBy); err != nil {
			tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
			failures = append(failures, BulkDeleteProgramFailure{
				ID:    id,
				Error: fmt.Sprintf("failed to delete: %v", err),
			})
			continue
		}

		if program.IsPublished() {
			if err := s.emitOutboxEvent(ctx, tx, domain.OutboxEventTypeProgramDelete, program); err != nil {
				tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
				failures = append(failures, BulkDeleteProgramFailure{
					ID:    id,
					Error: fmt.Sprintf("failed to emit event: %v", err),
				})
				continue
			}
		}

		tx.Exec(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName))
		deleted = append(deleted, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, []BulkDeleteProgramFailure{{Error: "failed to commit transaction"}}
	}

	return deleted, failures
}

func (s *Service) emitOutboxEvent(ctx context.Context, tx pgx.Tx, eventType domain.OutboxEventType, program *domain.Program) error {
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
