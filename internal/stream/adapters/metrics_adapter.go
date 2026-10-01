package adapters

import (
	"context"
	"fmt"
	"time"

	jobPublic "efficient-request-queueing-for-llm-inference/internal/job/public"
	metricsPublic "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
)

type MetricsAdapter struct {
	recorder  metricsPublic.StreamMetricsRecorder
	jobReader jobPublic.JobReader
}

var _ domain.MetricsRecorder = MetricsAdapter{}
var _ domain.JobAcceptedAtReader = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublic.StreamMetricsRecorder, jobReader jobPublic.JobReader) MetricsAdapter {
	return MetricsAdapter{recorder: recorder, jobReader: jobReader}
}

func (a MetricsAdapter) ObserveStreamDelivery(d time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveStreamDelivery(d)
	}
}

func (a MetricsAdapter) ObserveJobEndToEnd(d time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveJobEndToEnd(d)
	}
}

func (a MetricsAdapter) GetJobAcceptedAt(ctx context.Context, jobID string) (time.Time, error) {
	if a.jobReader == nil {
		return time.Time{}, fmt.Errorf("job reader must not be nil")
	}
	snapshot, err := a.jobReader.GetByID(ctx, jobID)
	if err != nil {
		return time.Time{}, err
	}
	return snapshot.CreatedAt, nil
}
