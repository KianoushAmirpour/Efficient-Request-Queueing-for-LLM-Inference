package application

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/admission"
	admissionAdapters "efficient-request-queueing-for-llm-inference/internal/admission/adapters"
	"efficient-request-queueing-for-llm-inference/internal/auth"
	userAdapter "efficient-request-queueing-for-llm-inference/internal/auth/adapters"
	healthchecker "efficient-request-queueing-for-llm-inference/internal/health_checker"
	"efficient-request-queueing-for-llm-inference/internal/inference"
	inferenceAdapters "efficient-request-queueing-for-llm-inference/internal/inference/adapters"
	inferenceserver "efficient-request-queueing-for-llm-inference/internal/inference_server"
	inferenceServerAdapters "efficient-request-queueing-for-llm-inference/internal/inference_server/adapters"
	"efficient-request-queueing-for-llm-inference/internal/job"
	jobAdapters "efficient-request-queueing-for-llm-inference/internal/job/adapters"
	"efficient-request-queueing-for-llm-inference/internal/observability/metrics"
	"efficient-request-queueing-for-llm-inference/internal/recovery"
	recoveryAdapters "efficient-request-queueing-for-llm-inference/internal/recovery/adapters"
	"efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerAdapters "efficient-request-queueing-for-llm-inference/internal/scheduler/adapters"
	"efficient-request-queueing-for-llm-inference/internal/stream"
	streamAdapters "efficient-request-queueing-for-llm-inference/internal/stream/adapters"
	"efficient-request-queueing-for-llm-inference/internal/user"
	userAdapters "efficient-request-queueing-for-llm-inference/internal/user/adapters"
	"efficient-request-queueing-for-llm-inference/internal/worker"
	workerAdapters "efficient-request-queueing-for-llm-inference/internal/worker/adapters"
)

func composeModules(
	pgPool *pgxpool.Pool,
	redisClient *redis.Client,
	appCfg AppConfig,
	logger *slog.Logger,
) (*Registry, error) {
	if appCfg.MetricsCfg == nil {
		return nil, fmt.Errorf("metrics configuration must not be nil")
	}
	metricsModule, err := metrics.NewMetricsModule(
		metrics.MetricsDeps{},
		metrics.MetricsConfig{
			Namespace: appCfg.MetricsCfg.Namespace,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	userModule, err := user.NewUserModule(
		user.UserDeps{
			PGPool:      pgPool,
			RedisClient: redisClient,
		},
		user.UserConfig{},
		logger,
	)
	if err != nil {
		return nil, err
	}
	userAdapter := userAdapter.NewUserAdapter(userModule.UserService)
	authModule, err := auth.NewAuthenticationModule(
		auth.AuthenticationDeps{
			PGPool:      pgPool,
			RedisClient: redisClient,
			UserService: userAdapter,
		},
		*appCfg.AuthCfg,
		logger,
	)
	if err != nil {
		return nil, err
	}
	tokenServiceAdapter := inferenceAdapters.NewTokenAuthenticatorAdapter(authModule.AuthTokenManager)
	tokenAdminServiceAdapter := userAdapters.NewTokenAuthenticatorAdapter(authModule.AuthTokenManager)
	streamTokenAdapter := streamAdapters.NewTokenAuthenticatorAdapter(authModule.AuthTokenManager)
	if err := userModule.SetTokenService(tokenAdminServiceAdapter); err != nil {
		return nil, err
	}

	userAdmAdapter := admissionAdapters.NewUserPolicyAdapter(userModule.UserService)
	admissionMetricsAdapter := admissionAdapters.NewMetricsAdapter(metricsModule.AdmissionMetrics)

	admissionModule, err := admission.NewAdmissionModule(
		admission.AdmissionDeps{
			RedisClient:      redisClient,
			UserPolicyReader: userAdmAdapter,
			Metrics:          admissionMetricsAdapter,
		},
		*appCfg.AdmissionCfg,
		logger,
	)
	if err != nil {
		return nil, err
	}

	userJobAdapter := jobAdapters.NewUserPolicyAdapter(userModule.UserService)
	jobModule, err := job.NewJobModule(
		job.JobDeps{
			PGPool:           pgPool,
			UserPolicyReader: userJobAdapter,
		},
		*appCfg.JobCfg,
		logger,
	)
	if err != nil {
		return nil, err
	}

	jobAdapter := inferenceAdapters.NewJobCreatorAdapter(jobModule.JobCreator)
	admissionAdapter := inferenceAdapters.NewRequestAdmitterAdapter(admissionModule.AdmissionService)

	schedulerModule, err := scheduler.NewSchedulerModule(
		scheduler.SchedulerDeps{
			RedisClient:  redisClient,
			LeaseTimeout: appCfg.WorkerCfg.Worker.LeaseTimeout,
			Metrics:      schedulerAdapters.NewMetricsAdapter(metricsModule.SchedulerMetrics),
		},
		appCfg.SchedulerCfg.Scheduler,
		logger,
	)
	if err != nil {
		return nil, err
	}
	queueServiceAdapter := inferenceAdapters.NewJobEnqueuerAdapter(schedulerModule.QueueService)

	inferenceModule, err := inference.NewInferenceModule(
		inference.InferenceDeps{
			RedisClient:            redisClient,
			TokenValidationService: tokenServiceAdapter,
			AdmissionService:       admissionAdapter,
			JobService:             jobAdapter,
			QueueService:           queueServiceAdapter,
			Metrics:                inferenceAdapters.NewMetricsAdapter(metricsModule.WorkloadMetrics),
		},
		appCfg.InferenceCfg.Inference,
		logger,
	)
	if err != nil {
		return nil, err
	}

	inferenceEngineModule, err := inferenceserver.NewInferenceEngineModule(
		inferenceserver.InferenceEngineDeps{Metrics: inferenceServerAdapters.NewMetricsAdapter(metricsModule.LLMMetrics)},
		appCfg.InferenceServerCfg.InferenceServer,
		logger,
	)
	if err != nil {
		return nil, err
	}

	jobClaimerAdapter := workerAdapters.NewJobClaimerAdapter(schedulerModule.JobClaimerService)
	jobRepoAdapter := workerAdapters.NewJobRepositoryAdapter(jobModule.JobCreator)
	inferenceAdapter := workerAdapters.NewInferenceEngineAdapter(inferenceEngineModule.InferenceEngine)
	streamMetricsAdapter := streamAdapters.NewMetricsAdapter(metricsModule.StreamMetrics, jobModule.JobReader)

	streamModule, err := stream.NewStreamModule(
		stream.StreamDeps{
			RedisClient:            redisClient,
			TokenValidationService: streamTokenAdapter,
			Metrics:                streamMetricsAdapter,
			JobAcceptedAtReader:    streamMetricsAdapter,
		},
		stream.StreamConfig{},
		logger,
	)
	if err != nil {
		return nil, err
	}
	jobRecoveryAdapter := recoveryAdapters.NewJobAdapter(jobModule.JobReader, jobModule.JobCreator)
	idempotencyAdapter := recoveryAdapters.NewIdempotencyStatusAdapter(inferenceModule.IdompService)

	recoveryModule, err := recovery.NewRecoveryModule(recovery.RecoveryDeps{
		Queue:       recoveryAdapters.NewSchedulerAdapter(schedulerModule.RecoveryService),
		Jobs:        jobRecoveryAdapter,
		Statuses:    jobRecoveryAdapter,
		Policies:    jobRecoveryAdapter,
		Idempotency: idempotencyAdapter,
		Events:      recoveryAdapters.NewEventsAdapter(streamModule.Publisher()),
		Metrics:     recoveryAdapters.NewMetricsAdapter(metricsModule.RecoveryMetrics, metricsModule.RecoveryRetryMetrics, metricsModule.JobTerminalMetrics),
	}, *appCfg.RecoveryCfg, logger)
	if err != nil {
		return nil, err
	}

	streamPublisherAdapter := workerAdapters.NewStreamPublisherAdapter(streamModule.Publisher())
	queueAdapter := workerAdapters.NewQueueAdapter(schedulerModule.QueueService)
	idempotencyStatusAdapter := workerAdapters.NewIdempotencyStatusAdapter(inferenceModule.IdompService)

	workerModule, err := worker.NewWorkerModule(
		worker.WorkerDeps{
			JobClaimer:         jobClaimerAdapter,
			JobService:         jobRepoAdapter,
			InferenceService:   inferenceAdapter,
			StreamPublisher:    streamPublisherAdapter,
			QueueService:       queueAdapter,
			IdempotencyUpdater: idempotencyStatusAdapter,
			Metrics:            workerAdapters.NewMetricsAdapter(metricsModule.WorkerMetrics, metricsModule.JobRetryMetrics, metricsModule.JobTerminalMetrics),
		},
		appCfg.WorkerCfg.Worker,
		logger,
	)
	if err != nil {
		return nil, err
	}
	healthCheckerModule, err := healthchecker.NewHealthCheckerModule(
		healthchecker.HealthCheckerDeps{
			PGPool:      pgPool,
			RedisClient: redisClient,
		},
		healthchecker.HealthCheckerConfig{Timeout: 5 * time.Second},
		logger,
	)
	if err != nil {
		return nil, err
	}
	moduleRegistry := &Registry{}
	moduleRegistry.Register(
		userModule,
		authModule,
		admissionModule,
		jobModule,
		schedulerModule,
		streamModule,
		inferenceModule,
		workerModule,
		recoveryModule,
		healthCheckerModule,
		metricsModule,
	)
	moduleRegistry.RegisterStartUpMetrics(metricsModule.HTTPMetrics)

	return moduleRegistry, nil
}
