package zerotrue

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the ZeroTrue API client.
type Client struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	maxRetries   int
	retryWaitMin time.Duration
	retryWaitMax time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the API base URL, stripping any trailing slash.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithRetryWaitMin sets the minimum retry wait duration.
func WithRetryWaitMin(d time.Duration) Option {
	return func(c *Client) {
		c.retryWaitMin = d
	}
}

// WithRetryWaitMax sets the maximum retry wait duration.
func WithRetryWaitMax(d time.Duration) Option {
	return func(c *Client) {
		c.retryWaitMax = d
	}
}

// NewClient creates a new ZeroTrue API client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if !strings.HasPrefix(apiKey, "zt_") || len(apiKey) < 4 {
		return nil, fmt.Errorf("zerotrue: invalid API key format, must start with 'zt_'")
	}

	c := &Client{
		apiKey:       apiKey,
		baseURL:      "https://api.zerotrue.app",
		httpClient:   &http.Client{Timeout: 5 * time.Minute},
		maxRetries:   3,
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// doRequest executes an HTTP request against the API with retry logic.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.baseURL + path

	// Read body once so it can be replayed on retries.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
	}

	maxAttempts := c.maxRetries + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 400 {
			if shouldRetry(resp.StatusCode) && attempt < maxAttempts-1 {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				wait := backoff(attempt, c.retryWaitMin, c.retryWaitMax)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
			return nil, parseErrorResponse(resp)
		}

		return resp, nil
	}

	return nil, fmt.Errorf("zerotrue: request failed after %d attempts", maxAttempts)
}
