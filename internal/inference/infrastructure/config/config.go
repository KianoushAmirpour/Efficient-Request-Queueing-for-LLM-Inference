package config

import "time"

type InferenceConfig struct {
	CoalescingTTL           time.Duration `yaml:"coalescing_ttl"`
	IdempotencyTTL          time.Duration `yaml:"idempotency_ttl"`
	CompletedIdempotencyTTL time.Duration `yaml:"completed_idempotency_ttl"`
}

type Config struct {
	Inference InferenceConfig `yaml:"inference"`
}
