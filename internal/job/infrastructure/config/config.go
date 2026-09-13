package config

import (
	"fmt"
	"math"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/job/domain"
)

type RetryConfig struct {
	MaxAttempt int           `yaml:"max_attempt"`
	BaseDelay  time.Duration `yaml:"base_delay"`
	MaxDelay   time.Duration `yaml:"max_delay"`
}

type JobConfig struct {
	Tiers map[string]RetryConfig `yaml:"user_tiers"`
}

func BuildJobPolicyConfig(cfg JobConfig) (domain.JobPolicyConfig, error) {
	if len(cfg.Tiers) == 0 {
		return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: at least one user tier must be configured")
	}

	tiers := make(map[string]domain.RetryPolicy, len(cfg.Tiers))

	for tierName, retryCfg := range cfg.Tiers {
		if tierName == "" {
			return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: user tier name must not be empty")
		}
		if retryCfg.MaxAttempt <= 0 {
			return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: max_attempt must be greater than zero for tier %q", tierName)
		}
		if retryCfg.MaxAttempt > math.MaxUint8 {
			return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: max_attempt must be <= %d for tier %q", math.MaxUint8, tierName)
		}
		if retryCfg.BaseDelay <= 0 {
			return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: base_delay must be greater than zero for tier %q", tierName)
		}
		if retryCfg.MaxDelay < retryCfg.BaseDelay {
			return domain.JobPolicyConfig{}, fmt.Errorf("job configuration: max_delay must be greater than or equal to base_delay for tier %q", tierName)
		}
		tiers[tierName] = domain.RetryPolicy{
			MaxAttempts: retryCfg.MaxAttempt,
			BaseDelay:   retryCfg.BaseDelay,
			MaxDelay:    retryCfg.MaxDelay,
		}
	}

	return domain.JobPolicyConfig{
		Tiers: tiers,
	}, nil
}
