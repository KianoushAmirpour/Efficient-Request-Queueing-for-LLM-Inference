package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type AdmissionMetrics struct {
	duration   *promclient.HistogramVec
	rejections *promclient.CounterVec
}

var _ public.AdmissionMetricsRecorder = (*AdmissionMetrics)(nil)

func NewAdmissionMetrics(builder *MetricBuilder) (*AdmissionMetrics, error) {
	duration := builder.newHistogramVecWithBuckets("admission_duration_seconds", "Admission decision duration.", shortDurationBuckets)
	rejections := builder.newCounterVec("admission_rejections_total", "Requests rejected by admission.", "reason")
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register admission metrics: %w", err)
	}
	for _, reason := range []string{admissionTokenLimit, admissionModelNotAllowed, admissionRateLimited, admissionOther} {
		rejections.WithLabelValues(reason)
	}
	return &AdmissionMetrics{duration: duration, rejections: rejections}, nil
}

func (m *AdmissionMetrics) RequestFinished(d time.Duration) { observe(m.duration.WithLabelValues(), d) }
func (m *AdmissionMetrics) ObserveAdmissionTokenLimitRejection() {
	m.rejections.WithLabelValues(admissionTokenLimit).Inc()
}
func (m *AdmissionMetrics) ObserveAdmissionModelNotAllowedRejection() {
	m.rejections.WithLabelValues(admissionModelNotAllowed).Inc()
}
func (m *AdmissionMetrics) ObserveAdmissionRateLimitedRejection() {
	m.rejections.WithLabelValues(admissionRateLimited).Inc()
}
func (m *AdmissionMetrics) ObserveAdmissionOtherRejection() {
	m.rejections.WithLabelValues(admissionOther).Inc()
}
