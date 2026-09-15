package domain

import "errors"

var (
	ErrDuplicateJob = errors.New("duplicate job: a job with this ID already exists in the queue")
	ErrQueueFull    = errors.New("queue full: user has reached the maximum number of concurrent jobs allowed")
	ErrJobNotFound  = errors.New("job not found: the job with the specified ID does not exist in the queue")
)
