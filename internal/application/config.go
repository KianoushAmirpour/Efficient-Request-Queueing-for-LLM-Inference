package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"efficient-request-queueing-for-llm-inference/infra/config"
	postgres "efficient-request-queueing-for-llm-inference/infra/pg_db"
	"efficient-request-queueing-for-llm-inference/infra/redis"
	admissionCfg "efficient-request-queueing-for-llm-inference/internal/admission/infrastructure/config"
	authCfg "efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/config"
	inferenceCfg "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
	inferenceServerCfg "efficient-request-queueing-for-llm-inference/internal/inference_server/infrastructure/config"
	jobCfg "efficient-request-queueing-for-llm-inference/internal/job/infrastructure/config"
	recoveryCfg "efficient-request-queueing-for-llm-inference/internal/recovery/infrastructure/config"
	schedulerCfg "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure/config"
	workerCfg "efficient-request-queueing-for-llm-inference/internal/worker/infrastructure/config"
)

type ServerSettings struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownGrace     time.Duration `yaml:"shutdown_grace"`
	HTTPDrainTimeout  time.Duration `yaml:"http_drain_timeout"`
}

type ServerConfig struct {
	Server ServerSettings `yaml:"server"`
}

type AppConfig struct {
	PostgresCfg        *postgres.PostgresConfig
	RedisCfg           *redis.RedisConfig
	AdmissionCfg       *admissionCfg.AdmissionConfig
	AuthCfg            *authCfg.AuthConfig
	ServerCfg          *ServerConfig
	JobCfg             *jobCfg.JobConfig
	WorkerCfg          *workerCfg.PoolConfig
	SchedulerCfg       *schedulerCfg.Config
	InferenceCfg       *inferenceCfg.Config
	RecoveryCfg        *recoveryCfg.RecoveryConfig
	InferenceServerCfg *inferenceServerCfg.Config
}

func newLoaderFromEnv() (*config.Loader, error) {
	cfgPath := os.Getenv("CFG_PATH")
	if cfgPath == "" {
		cfgPath = "configs"
	}

	absPath, err := filepath.Abs(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	return config.NewLoader(absPath), nil
}

func loadConfigs(loader *config.Loader) (AppConfig, error) {
	envErr := requireEnvs(
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"REDIS_PASSWORD",
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"JWT_ACCESS_SECRET",
		"OPENAI_API",
	)
	if envErr != nil {
		return AppConfig{}, envErr
	}

	pgCfg, err := config.LoadInto[postgres.PostgresConfig](loader, "postgres.yml")
	if err != nil {
		return AppConfig{}, err
	}
	pgCfg.Password = os.Getenv("POSTGRES_PASSWORD")
	pgCfg.User = os.Getenv("POSTGRES_USER")

	rdsCfg, err := config.LoadInto[redis.RedisConfig](loader, "redis.yml")
	if err != nil {
		return AppConfig{}, err
	}

	admCfg, err := config.LoadInto[admissionCfg.AdmissionConfig](loader, "admission.yml")
	if err != nil {
		return AppConfig{}, err
	}

	authConfig, err := config.LoadInto[authCfg.AuthConfig](loader, "auth.yml")
	if err != nil {
		return AppConfig{}, err
	}
	authConfig.GitHub.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	authConfig.GitHub.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	authConfig.JwtToken.JWTAccessSecret = os.Getenv("JWT_ACCESS_SECRET")

	serverCfg, err := config.LoadInto[ServerConfig](loader, "server.yml")
	if err != nil {
		return AppConfig{}, err
	}

	jobConfig, err := config.LoadInto[jobCfg.JobConfig](loader, "job.yml")
	if err != nil {
		return AppConfig{}, err
	}

	workerConfig, err := config.LoadInto[workerCfg.PoolConfig](loader, "worker.yml")
	if err != nil {
		return AppConfig{}, err
	}
	schedulerConfig, err := config.LoadInto[schedulerCfg.Config](loader, "scheduler.yml")
	if err != nil {
		return AppConfig{}, err
	}
	inferenceConfig, err := config.LoadInto[inferenceCfg.Config](loader, "inference.yml")
	if err != nil {
		return AppConfig{}, err
	}
	recoveryConfig, err := config.LoadInto[recoveryCfg.RecoveryConfig](loader, "recovery.yml")
	if err != nil {
		return AppConfig{}, err
	}

	inferenceEngineCfg, err := config.LoadInto[inferenceServerCfg.Config](loader, "inference_server.yml")
	if err != nil {
		return AppConfig{}, err
	}
	inferenceEngineCfg.InferenceServer.APIKey = os.Getenv("OPENAI_API")

	return AppConfig{
		PostgresCfg:        pgCfg,
		RedisCfg:           rdsCfg,
		AdmissionCfg:       admCfg,
		AuthCfg:            authConfig,
		ServerCfg:          serverCfg,
		JobCfg:             jobConfig,
		WorkerCfg:          workerConfig,
		SchedulerCfg:       schedulerConfig,
		InferenceCfg:       inferenceConfig,
		RecoveryCfg:        recoveryConfig,
		InferenceServerCfg: inferenceEngineCfg,
	}, nil
}

func requireEnvs(names ...string) error {
	var missing []string
	for _, n := range names {
		if os.Getenv(n) == "" {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}
