package retry

import (
	"context"
	"fmt"
	"time"
)

type RetryEvent struct {
	Attempt     int
	MaxAttempts int
	Err         error
	NextDelay   time.Duration
	WillRetry   bool
}

type RetryPolicy struct {
	MaxAttempts     int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	ShouldRetry     func(error) bool
	OnAttemptFailed func(RetryEvent)
}

func DoWithRetry[T any](
	ctx context.Context,
	policy RetryPolicy,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	if policy.MaxAttempts <= 0 {
		return zero, fmt.Errorf("retry policy: max attempts must be greater than zero")
	}

	var lastErr error

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		result, err := fn(ctx)

		if err == nil {
			return result, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		shouldRetry := policy.ShouldRetry == nil || policy.ShouldRetry(err)
		willRetry := shouldRetry && attempt < policy.MaxAttempts

		var nextDelay time.Duration
		if willRetry {
			nextDelay = retryDelay(policy.BaseDelay, policy.MaxDelay, attempt)
		}

		if policy.OnAttemptFailed != nil {
			policy.OnAttemptFailed(RetryEvent{
				Attempt:     attempt,
				MaxAttempts: policy.MaxAttempts,
				Err:         err,
				NextDelay:   nextDelay,
				WillRetry:   willRetry,
			})
		}

		if !willRetry {
			break
		}

		if err := sleepWithContext(ctx, nextDelay); err != nil {
			return zero, fmt.Errorf("retry interrupted: %w", err)
		}
	}

	return zero, fmt.Errorf(
		"operation failed after %d attempts: %w",
		policy.MaxAttempts,
		lastErr,
	)
}

func retryDelay(
	baseDelay time.Duration,
	maxDelay time.Duration,
	attempt int,
) time.Duration {
	if baseDelay <= 0 || attempt <= 0 {
		return 0
	}

	delay := baseDelay << (attempt - 1)

	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}

	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
