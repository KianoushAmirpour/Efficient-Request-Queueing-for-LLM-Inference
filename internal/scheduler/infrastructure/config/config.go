package config

import "time"

type SchedulerConfig struct {
	QueueCapacity     int           `yaml:"queue_capacity"`
	IdempotencyKeyTTL time.Duration `yaml:"idempotency_key_ttl"`
}

type Config struct {
	Scheduler SchedulerConfig `yaml:"scheduler"`
}
