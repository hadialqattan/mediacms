-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (type, payload, program_id)
VALUES ($1, $2, $3)
RETURNING *;
