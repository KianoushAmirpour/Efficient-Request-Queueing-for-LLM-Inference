-- +goose Up
-- +goose StatementBegin
CREATE TABLE oauth_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id varchar(36) NOT NULL UNIQUE,
    
    provider VARCHAR(64) NOT NULL,
    provider_user_id VARCHAR(256) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_oauth_accounts_users
        FOREIGN KEY (user_id)
        REFERENCES users(user_id)
        ON DELETE CASCADE,

    CONSTRAINT uq_provider_provider_user_id
        UNIQUE(provider, provider_user_id)
)
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE oauth_accounts;
-- +goose StatementEnd
