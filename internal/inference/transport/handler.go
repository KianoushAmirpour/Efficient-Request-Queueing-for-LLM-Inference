package transport

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	"efficient-request-queueing-for-llm-inference/internal/inference/usecase"
)

type SubmitInferenceHandler struct {
	SubmitInferenceUseCase *usecase.SubmitInferenceUseCase
	AccessTokenValidator   ports.AccessTokenValidator
	logger                 *slog.Logger
}

func NewSubmitInferenceHandler(
	inferenceUsecase *usecase.SubmitInferenceUseCase,
	accessTokenValidator ports.AccessTokenValidator,
	logger *slog.Logger,
) *SubmitInferenceHandler {
	return &SubmitInferenceHandler{
		SubmitInferenceUseCase: inferenceUsecase,
		AccessTokenValidator:   accessTokenValidator,
		logger:                 logger,
	}
}

func (s *SubmitInferenceHandler) HandleUserRequests(c *gin.Context) {

	var userRequest InferenceRequest
	err := c.ShouldBindJSON(&userRequest)
	if err != nil {
		_ = c.Error(err)
		return
	}

	incomingTask := &domain.InferenceInput{Prompt: userRequest.Prompt, Model: userRequest.Model}
	userID := c.GetString("userID")
	idempotencyHeader := c.GetString("Idempotency-Header")

	jobID, err := s.SubmitInferenceUseCase.Submit(c.Request.Context(), incomingTask, userID, idempotencyHeader)
	if err != nil {
		_ = c.Error(err)
		return
	}

	streamPath := fmt.Sprintf("/api/stream/%s", jobID)
	streamURL := fmt.Sprintf("%s://%s%s",
		c.Request.URL.Scheme,
		c.Request.Host,
		streamPath,
	)

	c.Header("Location", streamURL)
	c.JSON(http.StatusAccepted, InferenceResponse{
		JobID:     jobID,
		Message:   "Inference request accepted and queued for processing.",
		StreamURL: streamPath,
	})

}
