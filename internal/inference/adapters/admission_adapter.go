package adapters

import (
	"context"

	admissionPublicAPI "efficient-request-queueing-for-llm-inference/internal/admission/public"
	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
)

type RequestAdmitterAdapter struct {
	AdmissionService admissionPublicAPI.Admitter
}

var _ ports.RequestAdmitter = RequestAdmitterAdapter{}

func NewRequestAdmitterAdapter(admissionSvc admissionPublicAPI.Admitter) RequestAdmitterAdapter {
	return RequestAdmitterAdapter{AdmissionService: admissionSvc}
}

func (a RequestAdmitterAdapter) Admit(ctx context.Context, validatedTask *domain.InferenceRequest) (*domain.InferenceRequest, error) {

	reqToAdmit := admissionPublicAPI.AdmitInput{
		UserID: validatedTask.UserID,
		Prompt: validatedTask.Prompt,
		Model:  validatedTask.Model,
	}

	admittedOutput, err := a.AdmissionService.Admit(ctx, reqToAdmit)
	if err != nil {
		return nil, err
	}
	validatedTask.MaxOutputTokens = admittedOutput.MaxOutputTokens

	return validatedTask, nil
}
