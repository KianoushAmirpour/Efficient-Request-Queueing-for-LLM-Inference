package domain

import "errors"

var (
	ErrDuplicateJob = errors.New("duplicate job: a job with this ID already exists in the queue")
	ErrQueueFull    = errors.New("queue full: user has reached the maximum number of concurrent jobs allowed")
)
