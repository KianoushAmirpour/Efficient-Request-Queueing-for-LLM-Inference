package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
)

type SchedulerAdapter struct {
	service schedulerPublic.RecoveryService
}

var _ domain.Queue = SchedulerAdapter{}

func NewSchedulerAdapter(service schedulerPublic.RecoveryService) SchedulerAdapter {
	return SchedulerAdapter{service: service}
}

func (a SchedulerAdapter) ExpiredJobIDs(ctx context.Context, cutoffMs int64, limit int) ([]string, error) {
	return a.service.ExpiredJobIDs(ctx, cutoffMs, limit)
}

func (a SchedulerAdapter) RemoveIfExpired(ctx context.Context, jobID string, cutoffMs int64) (bool, error) {
	return a.service.RemoveIfExpired(ctx, jobID, cutoffMs)
}

func (a SchedulerAdapter) RequeueIfExpired(ctx context.Context, jobID, userID string, cutoffMs int64) (bool, error) {
	return a.service.RequeueIfExpired(ctx, jobID, userID, cutoffMs)
}

func (a SchedulerAdapter) CompletedJobIDs(ctx context.Context) ([]string, error) {
	return a.service.CompletedJobIDs(ctx)
}

func (a SchedulerAdapter) RemoveCompletedJob(ctx context.Context, jobID string) error {
	return a.service.RemoveCompletedJob(ctx, jobID)
}

func (a SchedulerAdapter) EnqueueIfAbsent(ctx context.Context, jobID, userID string, capacity int) (bool, error) {
	return a.service.EnqueueIfAbsent(ctx, jobID, userID, capacity)
}
