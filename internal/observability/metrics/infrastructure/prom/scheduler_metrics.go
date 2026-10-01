package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type SchedulerMetrics struct {
	wait *promclient.HistogramVec
}

var _ public.SchedulerMetricsRecorder = (*SchedulerMetrics)(nil)

func NewSchedulerMetrics(builder *MetricBuilder) (*SchedulerMetrics, error) {
	wait := builder.newHistogramVecWithBuckets("job_queue_wait_seconds", "Time from job enqueue until successful worker claim.", longDurationBuckets)
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register scheduler metrics: %w", err)
	}
	return &SchedulerMetrics{wait: wait}, nil
}

func (m *SchedulerMetrics) ObserveJobQueueWait(d time.Duration) { observe(m.wait.WithLabelValues(), d) }
