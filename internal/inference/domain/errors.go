package domain

import "errors"

var (
	ErrEmptyPrompt = errors.New("prompt can not be empty")
	ErrModelNotSet = errors.New("model must be set")
	ErrQueueFull   = errors.New("queue is full")
)
