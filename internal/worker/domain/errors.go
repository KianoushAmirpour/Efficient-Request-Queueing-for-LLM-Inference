package domain

import "errors"

var (
	ErrNoJobAvailable = errors.New("no job available in the queue")

	ErrJobProcessingFailed = errors.New("job processing failed")

	ErrWorkerShutdown = errors.New("worker shutdown")

	ErrPayloadFetchFailed = errors.New("payload fetch failed")

	ErrInferenceFailed = errors.New("inference generation failed")

	ErrTransientInference = errors.New("transient inference failure")
)
