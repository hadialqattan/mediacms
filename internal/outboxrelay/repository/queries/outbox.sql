-- name: GetPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE enqueued = false
ORDER BY created_at ASC
LIMIT 100;

-- name: MarkOutboxEventEnqueued :exec
UPDATE outbox_events
SET enqueued = true
WHERE id = $1;
