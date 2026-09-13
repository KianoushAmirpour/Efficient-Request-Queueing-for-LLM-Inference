-- +goose Up
-- +goose StatementBegin
CREATE TABLE job_results (
    job_id VARCHAR(36) PRIMARY KEY,
    response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_job_results_job
        FOREIGN KEY (job_id)
        REFERENCES jobs(job_id)
        ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE job_results;
-- +goose StatementEnd