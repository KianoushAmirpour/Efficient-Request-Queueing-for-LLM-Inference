package domain

type JobPayload struct {
	JobID           string
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}
