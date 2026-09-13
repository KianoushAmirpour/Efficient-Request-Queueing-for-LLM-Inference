-- +goose Up
-- +goose StatementBegin

CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TYPE user_tier AS ENUM ('free', 'premium');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL UNIQUE,
    role user_role NOT NULL DEFAULT 'user',
    tier user_tier NOT NULL DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE users;
DROP TYPE user_tier;
DROP TYPE user_role;

-- +goose StatementEnd