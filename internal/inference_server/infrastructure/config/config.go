package config

import "time"

type InferenceClient struct {
	APIKey                 string        `yaml:"-"`
	BaseURL                string        `yaml:"base_url"`
	MaxRetries             int           `yaml:"max_retries"`
	DialTimeout            time.Duration `yaml:"dial_timeout"`
	TLSHandshakeTimeout    time.Duration `yaml:"tls_handshake_timeout"`
	ResponseHeaderTimeout  time.Duration `yaml:"response_header_timeout"`
	StreamIdleTimeout      time.Duration `yaml:"stream_idle_timeout"`
	StreamTotalTimeout     time.Duration `yaml:"stream_total_timeout"`
	MaxIdleConnections     int           `yaml:"max_idle_connections"`
	MaxIdleConnectionsHost int           `yaml:"max_idle_connections_per_host"`
	ConnectionIdleTimeout  time.Duration `yaml:"connection_idle_timeout"`
}

type Config struct {
	InferenceServer InferenceClient `yaml:"inference_server"`
}
