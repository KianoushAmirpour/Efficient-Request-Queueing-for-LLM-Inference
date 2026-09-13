-- +goose Up
-- +goose StatementBegin

-- 1. Create the ENUM type for job status
CREATE TYPE job_status AS ENUM ('created','completed', 'failed');

-- 2. Create the jobs table
CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    job_id varchar(36) NOT NULL UNIQUE,
    user_id varchar(36) NOT NULL,
    status job_status NOT NULL,
    retry_counts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_jobs_users
        FOREIGN KEY (user_id)
        REFERENCES users(user_id)
        ON DELETE CASCADE
);

-- 3. Create index on user_id for faster lookups
CREATE INDEX idx_jobs_job_id ON jobs(job_id);

-- 4. Create index on status for queue operations
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_status_created_at ON jobs(status, created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 1. Drop the indexes
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_status_created_at;
DROP INDEX IF EXISTS idx_jobs_job_id;

-- 2. Drop the jobs table
DROP TABLE IF EXISTS jobs;

-- 3. Drop the ENUM type
DROP TYPE IF EXISTS job_status;

-- +goose StatementEnd