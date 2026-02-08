CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'editor')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('podcast', 'documentary')),
    language VARCHAR(5) NOT NULL CHECK (language IN ('ar', 'en')),
    duration_ms INTEGER NOT NULL,
    tags TEXT [] DEFAULT ARRAY [] :: TEXT [],
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    published_at TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id),
    published_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_by UUID REFERENCES users(id)
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL CHECK (type IN ('program.upsert', 'program.delete')),
    payload JSONB NOT NULL,
    enqueued BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    program_id UUID REFERENCES programs(id)
);