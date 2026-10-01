package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type LLMMetrics struct {
	timeToFirstToken *promclient.HistogramVec
	generation       *promclient.HistogramVec
}

var _ public.LLMGenerationMetricsRecorder = (*LLMMetrics)(nil)

func NewLLMMetrics(builder *MetricBuilder) (*LLMMetrics, error) {
	timeToFirstToken := builder.newHistogramVecWithBuckets("llm_time_to_first_token_seconds", "Time from generation start until the first non-empty output chunk.", ttftDurationBuckets)
	generation := builder.newHistogramVecWithBuckets("llm_generation_seconds", "End-to-end language model generation duration.", longDurationBuckets)
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register LLM metrics: %w", err)
	}
	return &LLMMetrics{timeToFirstToken: timeToFirstToken, generation: generation}, nil
}

func (m *LLMMetrics) ObserveTimeToFirstToken(d time.Duration) {
	observe(m.timeToFirstToken.WithLabelValues(), d)
}
func (m *LLMMetrics) ObserveGenerationDuration(d time.Duration) {
	observe(m.generation.WithLabelValues(), d)
}
