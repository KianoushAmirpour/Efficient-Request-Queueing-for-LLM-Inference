package config

import "time"

type Config struct {
	SweepInterval time.Duration `yaml:"sweep_interval"`
	BatchSize     int           `yaml:"batch_size"`
	OrphanGrace   time.Duration `yaml:"orphan_grace"`
	QueueCapacity int           `yaml:"queue_capacity"`
}

type RecoveryConfig struct {
	Config
}
