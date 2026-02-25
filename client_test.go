package zerotrue

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testAPIKey = "zt_testkey1234567890abcdef12345678"

// newTestClient starts an httptest server and returns a Client configured to use it.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(testAPIKey, WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient(testAPIKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", c.maxRetries)
	}
	if c.retryWaitMin != 1*time.Second {
		t.Errorf("retryWaitMin = %v, want 1s", c.retryWaitMin)
	}
	if c.retryWaitMax != 30*time.Second {
		t.Errorf("retryWaitMax = %v, want 30s", c.retryWaitMax)
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if c.httpClient.Timeout != 5*time.Minute {
		t.Errorf("httpClient.Timeout = %v, want 5m", c.httpClient.Timeout)
	}
	if c.apiKey != testAPIKey {
		t.Errorf("apiKey = %q, want %q", c.apiKey, testAPIKey)
	}
	if c.baseURL != "https://api.zerotrue.app" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.zerotrue.app")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 10 * time.Second}
	c, err := NewClient(testAPIKey,
		WithBaseURL("https://example.com/api/"),
		WithHTTPClient(customHTTP),
		WithTimeout(42*time.Second),
		WithMaxRetries(5),
		WithRetryWaitMin(2*time.Second),
		WithRetryWaitMax(60*time.Second),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.baseURL != "https://example.com/api" {
		t.Errorf("baseURL = %q, want trailing slash stripped", c.baseURL)
	}
	if c.httpClient != customHTTP {
		t.Error("httpClient was not set to custom client")
	}
	if c.httpClient.Timeout != 42*time.Second {
		t.Errorf("httpClient.Timeout = %v, want 42s", c.httpClient.Timeout)
	}
	if c.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want 5", c.maxRetries)
	}
	if c.retryWaitMin != 2*time.Second {
		t.Errorf("retryWaitMin = %v, want 2s", c.retryWaitMin)
	}
	if c.retryWaitMax != 60*time.Second {
		t.Errorf("retryWaitMax = %v, want 60s", c.retryWaitMax)
	}
}

func TestNewClient_InvalidAPIKey(t *testing.T) {
	_, err := NewClient("bad_key")
	if err == nil {
		t.Fatal("expected error for invalid API key")
	}
	if !strings.Contains(err.Error(), "zt_") {
		t.Errorf("error = %q, want mention of zt_ prefix", err.Error())
	}
}

func TestNewClient_EmptyAPIKey(t *testing.T) {
	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestClient_DoRequest_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	resp, err := c.doRequest(context.Background(), http.MethodGet, "/v1/test", nil, "")
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}

func TestClient_DoRequest_AuthHeader(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		want := "Bearer " + testAPIKey
		if got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
	})

	resp, err := c.doRequest(context.Background(), http.MethodGet, "/ping", nil, "")
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	resp.Body.Close()
}

func TestClient_DoRequest_ContentType(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Content-Type")
		if got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		w.WriteHeader(http.StatusOK)
	})

	resp, err := c.doRequest(context.Background(), http.MethodPost, "/v1/data",
		strings.NewReader(`{}`), "application/json")
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	resp.Body.Close()
}

func TestClient_DoRequest_ErrorResponse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Invalid API key"},"request_id":"req_123"}`))
	})

	_, err := c.doRequest(context.Background(), http.MethodGet, "/v1/secure", nil, "")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("error type = %T, want *AuthenticationError", err)
	}
	if authErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", authErr.StatusCode)
	}
	if authErr.RequestID != "req_123" {
		t.Errorf("RequestID = %q, want req_123", authErr.RequestID)
	}
}

func TestClient_DoRequest_ContextCancel(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.doRequest(ctx, http.MethodGet, "/slow", nil, "")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
