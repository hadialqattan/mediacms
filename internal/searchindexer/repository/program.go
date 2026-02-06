package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/searchindexer/port"
	"thmanyah.com/content-platform/internal/searchindexer/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type programReader struct {
	queries *sqlc.Queries
}

func NewProgramReader(pool *pgxpool.Pool) port.ProgramReader {
	return &programReader{
		queries: sqlc.New(pool),
	}
}

func (r *programReader) GetByID(ctx context.Context, id string) (*domain.Program, error) {
	program, err := r.queries.GetProgramByID(ctx, pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true})
	if err != nil {
		return nil, err
	}
	return toDomainProgram(program), nil
}

func (r *programReader) GetCategories(ctx context.Context, programID string) ([]domain.Category, error) {
	categories, err := r.queries.GetProgramCategories(ctx, pgtype.UUID{Bytes: uuid.MustParse(programID), Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Category, len(categories))
	for i, c := range categories {
		result[i] = domain.Category{
			ID:          uuid.UUID(c.ID.Bytes).String(),
			Name:        c.Name,
			Description: c.Description.String,
		}
	}
	return result, nil
}

func toDomainProgram(p sqlc.Program) *domain.Program {
	program := &domain.Program{
		ID:          uuid.UUID(p.ID.Bytes).String(),
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description.String,
		Type:        domain.ProgramType(p.Type),
		Language:    domain.ProgramLanguage(p.Language),
		DurationMs:  int(p.DurationMs),
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
	if p.SourceID.Valid {
		sourceID := uuid.UUID(p.SourceID.Bytes).String()
		program.SourceID = &sourceID
	}
	if p.UpdatedBy.Valid {
		updatedBy := uuid.UUID(p.UpdatedBy.Bytes).String()
		program.UpdatedBy = &updatedBy
	}

	return program
}
