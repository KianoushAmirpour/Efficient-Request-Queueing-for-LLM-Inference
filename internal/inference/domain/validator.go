package domain

import (
	"strings"
)

func NormalizeRequest(task *InferenceInput) *InferenceInput {
	prompt := strings.TrimSpace(task.Prompt)
	model := strings.TrimSpace(task.Model)
	return &InferenceInput{
		Prompt: prompt,
		Model:  model,
	}
}

func ValidateInferenceInput(inferenceInput *InferenceInput) (*InferenceRequest, error) {

	normalizedInferenceInput := NormalizeRequest(inferenceInput)

	if normalizedInferenceInput.Prompt == "" {
		return nil, ErrEmptyPrompt
	}

	model := normalizedInferenceInput.Model
	if model == "" {
		return nil, ErrModelNotSet
	}

	return &InferenceRequest{
		Prompt: normalizedInferenceInput.Prompt,
		Model:  normalizedInferenceInput.Model,
	}, nil

}
