CREATE INDEX idx_programs_deleted_at_created_at ON programs (deleted_at, created_at DESC);

CREATE INDEX idx_outbox_events_enqueued_created_at ON outbox_events (enqueued, created_at ASC);