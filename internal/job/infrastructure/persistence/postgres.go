package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"efficient-request-queueing-for-llm-inference/internal/job/domain"
	"efficient-request-queueing-for-llm-inference/internal/shared/tx/postgres"
)

type JobRepository struct {
	Db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{Db: db}
}

func (r *JobRepository) Save(ctx context.Context, job *domain.Job) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		INSERT INTO jobs (job_id, user_id, status, retry_counts)
		VALUES ($1, $2, $3::job_status, $4)
	`
	_, err := db.Exec(ctx, query,
		job.JobID,
		job.UserID,
		string(job.Status),
		job.CurrentAttempt,
	)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

func (r *JobRepository) SaveRequest(ctx context.Context, jobReq *domain.JobPayload) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		INSERT INTO job_requests (job_id, model, prompt, max_tokens, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := db.Exec(ctx, query, jobReq.JobID, jobReq.Model, jobReq.Prompt, jobReq.MaxOutputTokens)
	if err != nil {
		return fmt.Errorf("save job request: %w", err)
	}
	return nil
}

func (r *JobRepository) PayloadByID(ctx context.Context, jobID string) (*domain.JobPayload, error) {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		SELECT job_id, model, prompt, max_tokens
		FROM job_requests
		WHERE job_id = $1
	`
	var jobPayload domain.JobPayload
	err := db.QueryRow(ctx, query, jobID).Scan(
		&jobPayload.JobID,
		&jobPayload.Model,
		&jobPayload.Prompt,
		&jobPayload.MaxOutputTokens,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("get job by id: %w", err)
	}

	return &jobPayload, nil
}

func (r *JobRepository) JobByID(ctx context.Context, jobID string) (*domain.Job, error) {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		SELECT job_id, user_id, status, retry_counts
		FROM jobs
		WHERE job_id = $1
	`
	var job domain.Job
	err := db.QueryRow(ctx, query, jobID).Scan(
		&job.JobID,
		&job.UserID,
		&job.Status,
		&job.CurrentAttempt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("find job by id: %w", err)
	}

	return &job, nil
}

func (r *JobRepository) PendingOrphans(ctx context.Context, before time.Time, limit int) ([]*domain.Job, error) {
	db := postgres.ExtractDB(ctx, r.Db)
	rows, err := db.Query(ctx, `SELECT job_id, user_id, status, retry_counts, created_at FROM jobs WHERE status = 'created' AND created_at < $1 ORDER BY created_at LIMIT $2`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending orphan jobs: %w", err)
	}
	defer rows.Close()
	var jobs []*domain.Job
	for rows.Next() {
		var job domain.Job
		if err := rows.Scan(&job.JobID, &job.UserID, &job.Status, &job.CurrentAttempt, &job.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pending orphan job: %w", err)
		}
		jobs = append(jobs, &job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending orphan jobs: %w", err)
	}
	return jobs, nil
}

func (r *JobRepository) UpdateStatus(ctx context.Context, jobID string, status domain.Status, retryCount int) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		UPDATE jobs
		SET status = $2::job_status, retry_counts = $3, updated_at = NOW()
		WHERE job_id = $1
	`
	tag, err := db.Exec(ctx, query, jobID, domain.Status(status), retryCount)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrJobNotFound
	}

	return nil
}
