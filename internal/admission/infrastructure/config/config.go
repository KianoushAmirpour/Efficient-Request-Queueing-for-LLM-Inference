package config

import (
	_ "embed"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
)

type RateLimit struct {
	Capacity   int           `yaml:"capacity"`
	RefillRate int           `yaml:"refill_rate"`
	Ttl        time.Duration `yaml:"ttl"`
}

type RateLimitConfig struct {
	Free    RateLimit `yaml:"free"`
	Premium RateLimit `yaml:"premium"`
}

type ModelInfo struct {
	Name          string `yaml:"name"`
	ContextLength int    `yaml:"context_length"`
}

type TierConfig struct {
	MaxInputTokens  int         `yaml:"max_input_tokens"`
	MaxOutputTokens int         `yaml:"max_output_tokens"`
	Models          []ModelInfo `yaml:"models"`
}

type AdmissionConfig struct {
	RateLimit RateLimitConfig       `yaml:"rate_limit"`
	Tiers     map[string]TierConfig `yaml:"user_tiers"`
}

func BuildAdmissionPolicyConfig(cfg AdmissionConfig) domain.AdmissionPolicyConfig {
	tiers := make(map[string]domain.TierPolicy, len(cfg.Tiers))

	for tierName, tierCfg := range cfg.Tiers {

		allowedModels := make(map[string]struct{}, len(tierCfg.Models))
		modelContextLengths := make(map[string]int, len(tierCfg.Models))

		for _, model := range tierCfg.Models {
			allowedModels[model.Name] = struct{}{}
			modelContextLengths[model.Name] = model.ContextLength
		}

		tiers[tierName] = domain.TierPolicy{
			MaxInputTokens:      tierCfg.MaxInputTokens,
			MaxOutputTokens:     tierCfg.MaxOutputTokens,
			AllowedModels:       allowedModels,
			ModelContextLengths: modelContextLengths,
		}
	}

	return domain.AdmissionPolicyConfig{
		Tiers: tiers,
	}
}
