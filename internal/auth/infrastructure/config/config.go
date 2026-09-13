package config

import (
	"time"
)

type GitHubConfig struct {
	GitHubClientID     string `yaml:"-"`
	GitHubClientSecret string `yaml:"-"`
	GitHubCallbackURL  string `yaml:"callback_url"`
}

type JwtTokenConfig struct {
	JWTAccessSecret string        `yaml:"-"`
	JWTIssuer       string        `yaml:"issuer"`
	JWTAccessExpiry time.Duration `yaml:"access_expiry"`
	JWTAudience     string        `yaml:"audience"`
	JWTSubject      string        `yaml:"subject"`
}

type HttpClientConfig struct {
	TransportDialTimeout  time.Duration `yaml:"dial_timeout"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	MaxConnsPerHost       int           `yaml:"max_conns_per_host"`
	ConnectionIdleTimeout time.Duration `yaml:"idle_connection_timeout"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`
	RetryCount            int           `yaml:"retry_count"`
	RetryBaseDelay        time.Duration `yaml:"retry_base_delay"`
}

type AuthConfig struct {
	GitHub     GitHubConfig     `yaml:"github"`
	JwtToken   JwtTokenConfig   `yaml:"jwt"`
	HttpClient HttpClientConfig `yaml:"http_client"`
}
