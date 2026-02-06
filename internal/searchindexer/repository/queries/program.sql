-- name: GetProgramByID :one
SELECT * FROM programs WHERE id = $1;

-- name: GetProgramCategories :many
SELECT c.* FROM categories c
JOIN categorized_as ca ON c.id = ca.category_id
WHERE ca.program_id = $1;
