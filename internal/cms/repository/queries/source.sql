-- name: CreateSource :one
INSERT INTO sources (type, metadata)
VALUES ($1, $2)
RETURNING *;

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = $1;
