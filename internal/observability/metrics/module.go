package metrics

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/infrastructure/prom"
	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/transport"
)

type MetricsDeps struct{}

type MetricsConfig = config.MetricsConfig

type Module struct {
	HTTPMetrics          public.HTTPMetricsRecorder
	AdmissionMetrics     public.AdmissionMetricsRecorder
	WorkloadMetrics      public.WorkloadMetricsRecorder
	SchedulerMetrics     public.SchedulerMetricsRecorder
	WorkerMetrics        public.WorkerMetricsRecorder
	JobTerminalMetrics   public.JobTerminalMetricsRecorder
	JobRetryMetrics      public.JobRetryMetricsRecorder
	RecoveryRetryMetrics public.JobRecoveryRetryMetricsRecorder
	RecoveryMetrics      public.JobRecoveryMetricsRecorder
	LLMMetrics           public.LLMGenerationMetricsRecorder
	StreamMetrics        public.StreamMetricsRecorder
	handler              transport.MetricHandler
}

func NewMetricsModule(
	_ MetricsDeps,
	cfg MetricsConfig,
	logger *slog.Logger,
) (*Module, error) {
	if logger == nil {
		return nil, fmt.Errorf("metrics logger must not be nil")
	}
	registry := promclient.NewRegistry()
	builder, err := prom.NewMetricBuilder(registry, cfg.Namespace)
	if err != nil {
		return nil, fmt.Errorf("create prometheus metric builder: %w", err)
	}

	httpMetrics, err := prom.NewHTTPMetrics(builder)
	if err != nil {
		return nil, err
	}
	admissionMetrics, err := prom.NewAdmissionMetrics(builder)
	if err != nil {
		return nil, err
	}
	workloadMetrics, err := prom.NewWorkloadMetrics(builder)
	if err != nil {
		return nil, err
	}
	schedulerMetrics, err := prom.NewSchedulerMetrics(builder)
	if err != nil {
		return nil, err
	}
	workerMetrics, err := prom.NewWorkerMetrics(builder)
	if err != nil {
		return nil, err
	}
	jobMetrics, err := prom.NewJobLifecycleMetrics(builder)
	if err != nil {
		return nil, err
	}
	llmMetrics, err := prom.NewLLMMetrics(builder)
	if err != nil {
		return nil, err
	}
	streamMetrics, err := prom.NewStreamMetrics(builder)
	if err != nil {
		return nil, err
	}

	return &Module{
		HTTPMetrics:          httpMetrics,
		AdmissionMetrics:     admissionMetrics,
		WorkloadMetrics:      workloadMetrics,
		SchedulerMetrics:     schedulerMetrics,
		WorkerMetrics:        workerMetrics,
		JobTerminalMetrics:   jobMetrics,
		JobRetryMetrics:      jobMetrics,
		RecoveryRetryMetrics: jobMetrics,
		RecoveryMetrics:      prom.NewRecoveryMetrics(jobMetrics),
		LLMMetrics:           llmMetrics,
		StreamMetrics:        streamMetrics,
		handler:              transport.MetricHandler{Registry: registry},
	}, nil
}

func (m *Module) Name() string { return "Metrics" }

func (m *Module) RegisterRootRoutes(api *gin.RouterGroup) error {
	transport.RegisterMetricsRoutes(api, &m.handler)
	return nil
}
