package usecase

import (
	"context"
	"errors"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeAdmission = "ADMISSION_FAILED"

type AdmissionUsecase struct {
	users             domain.UserPolicyReader
	tokenPolicy       domain.PolicyEvaluator
	modelAccessPolicy domain.PolicyEvaluator
	rateLimitPolicy   domain.PolicyEvaluator
}

func NewAdmissionUseCase(
	users domain.UserPolicyReader,
	tokenPolicy domain.PolicyEvaluator,
	modelAccessPolicy domain.PolicyEvaluator,
	rateLimitPolicy domain.PolicyEvaluator,
) public.Admitter {
	return &AdmissionUsecase{
		users:             users,
		tokenPolicy:       tokenPolicy,
		modelAccessPolicy: modelAccessPolicy,
		rateLimitPolicy:   rateLimitPolicy,
	}
}

func (a *AdmissionUsecase) Admit(
	ctx context.Context,
	req public.AdmitInput,
) (public.AdmitOutput, error) {

	admissionRequest := domain.AdmissionRequest{
		UserID: req.UserID,
		Model:  req.Model,
		Prompt: req.Prompt,
	}

	userPolicy, err :=
		a.users.GetUserPolicy(
			ctx,
			req.UserID,
		)
	if err != nil {
		return public.AdmitOutput{}, sharederr.NewAppError(
			public.ErrCodePolicyFetchFailed,
			ErrTypeAdmission,
			err,
		)
	}

	policyCtx := domain.PolicyEvaluationContext{
		Request: admissionRequest,
		User:    userPolicy,
	}

	err = a.modelAccessPolicy.Evaluate(ctx, &policyCtx)
	if err != nil {
		return public.AdmitOutput{}, sharederr.EnsureAppError(err, public.ErrCodeModelNotAllowed, ErrTypeAdmission)
	}

	err = a.tokenPolicy.Evaluate(ctx, &policyCtx)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInputTooLarge):
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, public.ErrCodeInputToken, ErrTypeAdmission)

		case errors.Is(err, domain.ErrContextLengthExceeded):
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, public.ErrCodeCostExceeded, ErrTypeAdmission)

		default:
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, sharederr.ErrCodeInternal, ErrTypeAdmission)
		}
	}

	err = a.rateLimitPolicy.Evaluate(ctx, &policyCtx)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRateLimited):
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, public.ErrCodeRateLimited, ErrTypeAdmission)
		case errors.Is(err, domain.ErrCostExceedsCapacity):
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, public.ErrCodeCostExceeded, ErrTypeAdmission)
		default:
			return public.AdmitOutput{}, sharederr.EnsureAppError(err, sharederr.ErrCodeInternal, ErrTypeAdmission)
		}
	}
	return public.AdmitOutput{
		Admitted:        true,
		UserID:          admissionRequest.UserID,
		Prompt:          admissionRequest.Prompt,
		Model:           admissionRequest.Model,
		MaxOutputTokens: policyCtx.Metadata.MaxOutputTokens,
	}, nil
}
