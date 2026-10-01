package prom

import (
	"fmt"
	"strings"
)

const (
	admissionTokenLimit      = "token_limit"
	admissionModelNotAllowed = "model_not_allowed"
	admissionRateLimited     = "rate_limited"
	admissionOther           = "other"

	jobStatusSucceeded = "succeeded"
	jobStatusFailed    = "failed"

	retryModelFailure = "model_failure"
	retryLeaseExpired = "lease_expired"
	retryRecovery     = "recovery"

	recoveryRequeued = "requeued"
	recoveryFailed   = "failed"

	processingSuccess = "succeeded"
	processingFailed  = "failed"
)

func normalizeHTTPMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return method
	default:
		return "OTHER"
	}
}

func normalizeHTTPRoute(route string) string {
	route = strings.TrimPrefix(route, "/api")
	switch route {
	case "/request",
		"/health",
		"/metrics",
		"/stream/:jobID",
		"/auth/login/github",
		"/auth/github/callback",
		"/admin/register",
		"/admin/login",
		"/admin/users/:user_id/tier":
		return route
	default:
		return "other"
	}
}

func normalizeHTTPStatus(status int) string {
	if status < 100 || status > 599 {
		return "other"
	}
	return fmt.Sprintf("%dxx", status/100)
}
