package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type programRepo struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewProgramRepo(pool *pgxpool.Pool) port.ProgramRepo {
	return &programRepo{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *programRepo) WithTx(tx pgx.Tx) port.ProgramRepo {
	return &programRepo{
		pool:    r.pool,
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

func (r *programRepo) GetBySlug(ctx context.Context, slug string) (*domain.Program, error) {
	program, err := r.queries.GetProgramBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return r.domainProgram(program), nil
}

func (r *programRepo) List(ctx context.Context) ([]*domain.Program, error) {
	programs, err := r.queries.ListPrograms(ctx)
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

func (r *programRepo) AssignCategories(ctx context.Context, programID string, categoryIDs []string) error {
	uuids := make([]pgtype.UUID, len(categoryIDs))
	for i, id := range categoryIDs {
		uuids[i] = pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true}
	}

	return r.queries.AssignCategories(ctx, sqlc.AssignCategoriesParams{
		ProgramID: pgtype.UUID{Bytes: uuid.MustParse(programID), Valid: true},
		Column2:   uuids,
	})
}

func (r *programRepo) GetCategories(ctx context.Context, programID string) ([]domain.Category, error) {
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

func (r *programRepo) domainProgram(p sqlc.Program) *domain.Program {
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
