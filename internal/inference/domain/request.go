package domain

import (
	"strings"
)

type InferenceInput struct {
	Prompt string
	Model  string
}

type InferenceRequest struct {
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}

type InferenceJob struct {
	JobID          string
	UserID         string
	CurrentAttempt int
}

func (v *InferenceRequest) GenerateKey() string {
	var b strings.Builder
	b.Grow(len(v.UserID) + len(v.Prompt) + len(v.Model) + 50)

	b.WriteString(v.UserID)
	b.WriteByte(':')
	b.WriteString(v.Prompt)
	b.WriteByte(':')
	b.WriteString(v.Model)

	return b.String()
}
