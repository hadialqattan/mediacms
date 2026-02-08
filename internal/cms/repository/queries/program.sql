-- name: CreateProgram :one
INSERT INTO programs (slug, title, description, type, language, duration_ms, tags, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetProgramByID :one
SELECT * FROM programs WHERE id = $1;

-- name: ListPrograms :many
SELECT * FROM programs WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;


-- name: UpdateProgram :one
UPDATE programs
SET
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  type = COALESCE($4, type),
  language = COALESCE($5, language),
  duration_ms = COALESCE($6, duration_ms),
  tags = COALESCE($7, tags),
  updated_at = NOW(),
  updated_by = $8
WHERE id = $1
RETURNING *;

-- name: PublishProgram :one
UPDATE programs
SET published_at = NOW(), published_by = $2
WHERE id = $1
RETURNING *;

-- name: DeleteProgram :exec
UPDATE programs
SET deleted_at = NOW(), deleted_by = $2
WHERE id = $1;
