package public

import (
	"context"
)

type AdmitInput struct {
	UserID string
	Prompt string
	Model  string
}

type AdmitOutput struct {
	Admitted        bool
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}

type Admitter interface {
	Admit(ctx context.Context, admitInput AdmitInput) (AdmitOutput, error)
}
