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

type programRepo struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewProgramRepo(db sqlc.DBTX) port.ProgramRepo {
	return &programRepo{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *programRepo) WithTx(tx pgx.Tx) port.ProgramRepo {
	return &programRepo{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

func (r *programRepo) Create(ctx context.Context, params sqlc.CreateProgramParams) (*domain.Program, error) {
	program, err := r.queries.CreateProgram(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.domainProgram(program), nil
}

func (r *programRepo) GetByID(ctx context.Context, id string) (*domain.Program, error) {
	program, err := r.queries.GetProgramByID(ctx, pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true})
	if err != nil {
		return nil, err
	}

	return r.domainProgram(program), nil
}

func (r *programRepo) List(ctx context.Context, limit, offset int) ([]*domain.Program, error) {
	programs, err := r.queries.ListPrograms(ctx, sqlc.ListProgramsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Program, len(programs))
	for i, p := range programs {
		result[i] = r.domainProgram(p)
	}
	return result, nil
}

func (r *programRepo) Update(ctx context.Context, id string, params sqlc.UpdateProgramParams) (*domain.Program, error) {
	result, err := r.queries.UpdateProgram(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.domainProgram(result), nil
}

func (r *programRepo) Publish(ctx context.Context, id, publishedBy string) (*domain.Program, error) {
	program, err := r.queries.PublishProgram(ctx, sqlc.PublishProgramParams{
		ID:          pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true},
		PublishedBy: pgtype.UUID{Bytes: uuid.MustParse(publishedBy), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return r.domainProgram(program), nil
}

func (r *programRepo) Delete(ctx context.Context, id, deletedBy string) error {
	return r.queries.DeleteProgram(ctx, sqlc.DeleteProgramParams{
		ID:        pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true},
		DeletedBy: pgtype.UUID{Bytes: uuid.MustParse(deletedBy), Valid: true},
	})
}

func (r *programRepo) domainProgram(p sqlc.Program) *domain.Program {
	program := &domain.Program{
		ID:          uuid.UUID(p.ID.Bytes).String(),
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description.String,
		Type:        domain.ProgramType(p.Type),
		Language:    domain.ProgramLanguage(p.Language),
		DurationMs:  int(p.DurationMs),
		Tags:        p.Tags,
		CreatedAt:   p.CreatedAt.Time,
		CreatedBy:   uuid.UUID(p.CreatedBy.Bytes).String(),
	}

	if p.UpdatedAt.Valid {
		program.UpdatedAt = &p.UpdatedAt.Time
	}
	if p.PublishedAt.Valid {
		program.PublishedAt = &p.PublishedAt.Time
		publishedBy := uuid.UUID(p.PublishedBy.Bytes).String()
		program.PublishedBy = &publishedBy
	}
	if p.DeletedAt.Valid {
		program.DeletedAt = &p.DeletedAt.Time
		deletedBy := uuid.UUID(p.DeletedBy.Bytes).String()
		program.DeletedBy = &deletedBy
	}
	if p.UpdatedBy.Valid {
		updatedBy := uuid.UUID(p.UpdatedBy.Bytes).String()
		program.UpdatedBy = &updatedBy
	}

	return program
}
