package config

import "time"

type SchedulerConfig struct {
	QueueCapacity       int           `yaml:"queue_capacity"`
	IdempotencyKeyTTL   time.Duration `yaml:"idempotency_key_ttl"`
	ActiveUsersInterval time.Duration `yaml:"active_users_interval"`
}

type Config struct {
	Scheduler SchedulerConfig `yaml:"scheduler"`
}
