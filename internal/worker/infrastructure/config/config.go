package config

import "time"

type PoolConfig struct {
	Worker WorkerConfig `yaml:"worker"`
}

type WorkerConfig struct {
	WorkerCounts       int           `yaml:"num_workers"`
	PollInterval       time.Duration `yaml:"poll_interval"`
	LeaseTimeout       time.Duration `yaml:"lease_timeout"`
	Temperature        float32       `yaml:"temperature"`
	DefaultMaxAttempts int           `yaml:"default_max_attempts"`
	DefaultMaxDelay    time.Duration `yaml:"default_max_delay"`
	DefaultBaseDelay   time.Duration `yaml:"default_base_delay"`
}
