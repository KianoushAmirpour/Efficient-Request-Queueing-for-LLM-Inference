package job

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"efficient-request-queueing-for-llm-inference/internal/job/domain"
	"efficient-request-queueing-for-llm-inference/internal/job/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/job/infrastructure/persistence"
	"efficient-request-queueing-for-llm-inference/internal/job/public"
	"efficient-request-queueing-for-llm-inference/internal/job/usecase"
	txpostgres "efficient-request-queueing-for-llm-inference/internal/shared/tx/postgres"
)

type JobDeps struct {
	PGPool           *pgxpool.Pool
	UserPolicyReader domain.UserPolicyReader
}

type JobConfig = config.JobConfig

type Module struct {
	JobCreator public.JobService
	JobReader  public.JobReader
}

func NewJobModule(
	deps JobDeps,
	cfg JobConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.PGPool == nil {
		return nil, fmt.Errorf("job dependencies: postgres pool must not be nil")
	}
	if deps.UserPolicyReader == nil {
		return nil, fmt.Errorf("job dependencies: user policy reader must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("job logger must not be nil")
	}

	jobRepo := persistence.NewJobRepository(deps.PGPool)
	jobPolicyCfg, err := config.BuildJobPolicyConfig(cfg)
	if err != nil {
		return nil, err
	}
	uow := txpostgres.NewUnitOfWork(deps.PGPool, logger)

	service := usecase.NewJobUsecase(jobRepo, jobPolicyCfg, deps.UserPolicyReader, uow)
	return &Module{JobCreator: service, JobReader: service.(public.JobReader)}, nil
}

func (m *Module) Name() string { return "Job" }
