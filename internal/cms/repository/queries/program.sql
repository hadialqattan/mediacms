-- name: CreateProgram :one
INSERT INTO programs (slug, title, description, type, language, duration_ms, source_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetProgramByID :one
SELECT * FROM programs WHERE id = $1;

-- name: GetProgramBySlug :one
SELECT * FROM programs WHERE slug = $1;

-- name: ListPrograms :many
SELECT * FROM programs WHERE deleted_at IS NULL ORDER BY created_at DESC;

-- name: UpdateProgram :one
UPDATE programs
SET
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  type = COALESCE($4, type),
  language = COALESCE($5, language),
  duration_ms = COALESCE($6, duration_ms),
  updated_at = NOW(),
  updated_by = $7
WHERE id = $1
RETURNING *;

-- name: PublishProgram :one
UPDATE programs
SET published_at = NOW(), published_by = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProgram :exec
UPDATE programs
SET deleted_at = NOW(), deleted_by = $2
WHERE id = $1;

-- name: AssignCategories :exec
INSERT INTO categorized_as (program_id, category_id)
VALUES ($1, unnest($2::uuid[]))
ON CONFLICT (program_id, category_id) DO NOTHING;

-- name: RemoveCategories :exec
DELETE FROM categorized_as
WHERE program_id = $1 AND category_id = ANY($2::uuid[]);

-- name: GetProgramCategories :many
SELECT c.* FROM categories c
JOIN categorized_as ca ON c.id = ca.category_id
WHERE ca.program_id = $1;
