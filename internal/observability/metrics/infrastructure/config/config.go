package config

type MetricsConfig struct {
	Namespace string `yaml:"namespace"`
}

type Config struct {
	Metrics MetricsConfig `yaml:"metrics"`
}
