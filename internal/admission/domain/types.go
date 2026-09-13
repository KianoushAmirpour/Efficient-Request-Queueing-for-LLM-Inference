package domain

type TierPolicy struct {
	MaxInputTokens      int
	MaxOutputTokens     int
	AllowedModels       map[string]struct{}
	ModelContextLengths map[string]int
}

type AdmissionPolicyConfig struct {
	Tiers map[string]TierPolicy
}

type AdmissionRequest struct {
	UserID string
	Prompt string
	Model  string
}

type RequestMetadata struct {
	InputTokens     int
	MaxOutputTokens int
}

type UserPolicy struct {
	Tier string
}

type PolicyEvaluationContext struct {
	Request  AdmissionRequest
	Metadata RequestMetadata
	User     UserPolicy
}
