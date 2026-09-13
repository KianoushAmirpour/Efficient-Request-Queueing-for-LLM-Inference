package transport

type InferenceRequest struct {
	Prompt string `json:"prompt" binding:"required"`
	Model  string `json:"model" binding:"required"`
}

type InferenceResponse struct {
	JobID     string `json:"job_id"`
	Message   string `json:"message"`
	StreamURL string `json:"url"`
}
