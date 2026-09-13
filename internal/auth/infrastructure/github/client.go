package github

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/config"
)

type HTTPClient struct {
	clientID       string
	clientSecret   string
	callbackURL    string
	httpClient     *http.Client
	retryCount     int
	retryBaseDelay time.Duration
}

func NewHTTPClient(cfg config.AuthConfig) *HTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.HttpClient.TransportDialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		MaxIdleConns:          cfg.HttpClient.MaxIdleConnections,
		MaxConnsPerHost:       cfg.HttpClient.MaxConnsPerHost,
		IdleConnTimeout:       cfg.HttpClient.ConnectionIdleTimeout,
		TLSHandshakeTimeout:   cfg.HttpClient.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.HttpClient.ResponseHeaderTimeout,
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	return &HTTPClient{
		clientID:       cfg.GitHub.GitHubClientID,
		clientSecret:   cfg.GitHub.GitHubClientSecret,
		callbackURL:    cfg.GitHub.GitHubCallbackURL,
		httpClient:     httpClient,
		retryCount:     cfg.HttpClient.RetryCount,
		retryBaseDelay: cfg.HttpClient.RetryBaseDelay,
	}
}

type RetryPolicy struct {
	MaxRetries  int
	BaseDelay   time.Duration
	ShouldRetry func(status int, err error) bool
}

func (c *HTTPClient) AuthorizationURL(state string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "/login/oauth/authorize",
	}
	q := u.Query()
	q.Set("client_id", c.clientID)
	q.Set("redirect_uri", c.callbackURL)
	q.Set("scope", "read")
	q.Set("state", state)

	u.RawQuery = q.Encode()
	return u.String()
}

func (c *HTTPClient) doWithRetry(
	ctx context.Context,
	req *http.Request,
	policy RetryPolicy,
	decode func(*http.Response) error,
) error {

	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {

		resp, err := c.httpClient.Do(req)

		if err == nil {
			if !policy.ShouldRetry(resp.StatusCode, nil) {
				defer func() {
					_, err := io.Copy(io.Discard, resp.Body)
					if err != nil {
						lastErr = fmt.Errorf("failed to consume response body: %w", err)
					}
					_ = resp.Body.Close()
				}()

				if err := decode(resp); err != nil {
					return fmt.Errorf("github request decode status=%d: %w", resp.StatusCode, err)
				}
				return nil
			}

			_, err := io.Copy(io.Discard, resp.Body)
			if err != nil {
				return fmt.Errorf("failed to consume response body: %w", err)
			}
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("github request retryable: status=%d", resp.StatusCode)
		} else {
			if !policy.ShouldRetry(0, err) {
				return fmt.Errorf("github request failed: %w", err)
			}
			lastErr = err
		}

		if attempt < policy.MaxRetries {
			if err := sleepWithContext(ctx, policy.BaseDelay, attempt); err != nil {
				return fmt.Errorf("github API retry: %w", err)
			}
		}
	}

	lastErr = fmt.Errorf("github request failed after retries: %w", lastErr)
	return lastErr
}

func (c *HTTPClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	u := "https://github.com/login/oauth/access_token"

	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.callbackURL)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("github code exchange request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}

	policy := RetryPolicy{
		MaxRetries: 1,
		BaseDelay:  200 * time.Millisecond,
		ShouldRetry: func(status int, err error) bool {
			if err != nil {
				return isNetworkTransient(err)
			}
			return status == 503
		},
	}

	err = c.doWithRetry(ctx, req, policy, func(r *http.Response) error {

		if r.StatusCode != http.StatusOK {
			return fmt.Errorf("github token response: status=%d", r.StatusCode)
		}

		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			return fmt.Errorf("decode github token response: %w", err)
		}

		if resp.Error != "" {
			return fmt.Errorf("github oauth error: %s", resp.Error)
		}

		if resp.AccessToken == "" {
			return fmt.Errorf("missing access_token in github response")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return resp.AccessToken, nil
}

func (c *HTTPClient) GetUserInfo(ctx context.Context, token string) (*domain.OAuthUser, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.github.com/user",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("github fetch user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	var GitResp struct {
		GitHubUserID int64 `json:"id"`
	}

	policy := RetryPolicy{
		MaxRetries: c.retryCount,
		BaseDelay:  c.retryBaseDelay,
		ShouldRetry: func(status int, err error) bool {
			if err != nil {
				return isNetworkTransient(err)
			}
			return status == 502 || status == 503 || status == 504 || status == 429
		},
	}

	err = c.doWithRetry(ctx, req, policy, func(r *http.Response) error {

		if r.StatusCode != http.StatusOK {
			return fmt.Errorf("github token response: status=%d", r.StatusCode)
		}

		if err := json.NewDecoder(r.Body).Decode(&GitResp); err != nil {
			return fmt.Errorf("decode github token response: %w", err)
		}

		if GitResp.GitHubUserID == 0 || GitResp.GitHubUserID < 0 {
			return fmt.Errorf("invalid github user id in response")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	userID := strconv.FormatInt(GitResp.GitHubUserID, 10)

	return &domain.OAuthUser{
		ProviderUserID: userID,
		Provider:       "github",
	}, nil
}

func isNetworkTransient(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func sleepWithContext(ctx context.Context, base time.Duration, attempt int) error {
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))

	maxJitter := int64(delay / 5)

	var jitter time.Duration
	if maxJitter > 0 {
		bigMax := big.NewInt(maxJitter)

		bigJitter, err := rand.Int(rand.Reader, bigMax)
		if err != nil {
			return fmt.Errorf("generate jitter: %w", err)
		}

		jitter = time.Duration(bigJitter.Int64())
	}

	delay += jitter

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
