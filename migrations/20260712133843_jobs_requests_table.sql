-- +goose Up
-- +goose StatementBegin
CREATE TABLE job_requests (
    job_id VARCHAR(36) PRIMARY KEY,
    model VARCHAR(36) NOT NULL,
    prompt TEXT NOT NULL,
    max_tokens INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_job_requests_job
        FOREIGN KEY (job_id)
        REFERENCES jobs(job_id)
        ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE job_requests;
-- +goose StatementEnd