package domain

type JobPolicyConfig struct {
	Tiers map[string]RetryPolicy
}
