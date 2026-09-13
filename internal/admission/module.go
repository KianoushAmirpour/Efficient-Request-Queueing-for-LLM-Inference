package admission

import (
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	"efficient-request-queueing-for-llm-inference/internal/admission/domain/policy"
	"efficient-request-queueing-for-llm-inference/internal/admission/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/admission/infrastructure/ratelimit"
	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	"efficient-request-queueing-for-llm-inference/internal/admission/usecase"
)

type AdmissionDeps struct {
	RedisClient      *redis.Client
	UserPolicyReader domain.UserPolicyReader
}

type AdmissionConfig = config.AdmissionConfig

type admissionDependency struct {
	tokenPolicy       domain.PolicyEvaluator
	modelAccessPolicy domain.PolicyEvaluator
	rateLimitPolicy   domain.PolicyEvaluator
}

func buildAdmissionDeps(redisClient *redis.Client, admissionConfig AdmissionConfig) admissionDependency {
	tierRateLimits := map[string]ratelimit.TierRateLimit{
		"free": {
			Capacity: admissionConfig.RateLimit.Free.Capacity,
			FillRate: admissionConfig.RateLimit.Free.RefillRate,
			TTL:      admissionConfig.RateLimit.Free.Ttl,
		},
		"premium": {
			Capacity: admissionConfig.RateLimit.Premium.Capacity,
			FillRate: admissionConfig.RateLimit.Premium.RefillRate,
			TTL:      admissionConfig.RateLimit.Premium.Ttl,
		},
	}

	tokenBucketLimiter := ratelimit.NewRedisRateLimiter(
		redisClient,
		ratelimit.RateLimitConfig{Tiers: tierRateLimits},
	)

	rateLimitPolicy := policy.NewRateLimitPolicy(tokenBucketLimiter)

	admissionPolicy := config.BuildAdmissionPolicyConfig(admissionConfig)

	tokenPolicy := policy.NewTokenLimitPolicy(admissionPolicy)

	modelAccessPolicy := policy.NewModelAccessPolicy(admissionPolicy)

	return admissionDependency{
		tokenPolicy:       tokenPolicy,
		modelAccessPolicy: modelAccessPolicy,
		rateLimitPolicy:   rateLimitPolicy,
	}
}

type Module struct {
	AdmissionService public.Admitter
}

func NewAdmissionModule(
	deps AdmissionDeps,
	cfg AdmissionConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("admission dependencies: redis client must not be nil")
	}
	if deps.UserPolicyReader == nil {
		return nil, fmt.Errorf("admission dependencies: user policy reader must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("admission logger must not be nil")
	}
	if err := validateAdmissionConfig(cfg); err != nil {
		return nil, err
	}

	admissionModDeps := buildAdmissionDeps(deps.RedisClient, cfg)
	admissionUseCase := usecase.NewAdmissionUseCase(
		deps.UserPolicyReader,
		admissionModDeps.tokenPolicy,
		admissionModDeps.modelAccessPolicy,
		admissionModDeps.rateLimitPolicy,
	)

	return &Module{
		AdmissionService: admissionUseCase,
	}, nil
}

func validateAdmissionConfig(cfg AdmissionConfig) error {
	if err := validateRateLimit("free", cfg.RateLimit.Free); err != nil {
		return err
	}
	if err := validateRateLimit("premium", cfg.RateLimit.Premium); err != nil {
		return err
	}
	if len(cfg.Tiers) == 0 {
		return fmt.Errorf("admission configuration: at least one user tier is required")
	}
	for tierName, tier := range cfg.Tiers {
		if tierName == "" {
			return fmt.Errorf("admission configuration: tier name must not be empty")
		}
		if tier.MaxInputTokens <= 0 || tier.MaxOutputTokens <= 0 {
			return fmt.Errorf("admission configuration: tier %q token limits must be positive", tierName)
		}
		if len(tier.Models) == 0 {
			return fmt.Errorf("admission configuration: tier %q must define at least one model", tierName)
		}
		for _, model := range tier.Models {
			if model.Name == "" {
				return fmt.Errorf("admission configuration: tier %q contains an empty model name", tierName)
			}
			if model.ContextLength <= 0 {
				return fmt.Errorf("admission configuration: tier %q model %q context length must be positive", tierName, model.Name)
			}
		}
	}
	return nil
}

func validateRateLimit(name string, limit config.RateLimit) error {
	if limit.Capacity <= 0 || limit.RefillRate <= 0 || limit.Ttl <= 0 {
		return fmt.Errorf("admission configuration: %s rate limit capacity, refill rate, and TTL must be positive", name)
	}
	return nil
}

func (m *Module) Name() string { return "Admission" }
