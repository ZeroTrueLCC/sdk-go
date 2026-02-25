package zerotrue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{429, true},
		{400, false},
		{401, false},
		{404, false},
		{200, false},
		{301, false},
	}
	for _, tt := range tests {
		got := shouldRetry(tt.code)
		if got != tt.want {
			t.Errorf("shouldRetry(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestBackoff(t *testing.T) {
	min := 10 * time.Millisecond
	max := 5 * time.Second

	attempts := []int{0, 1, 2, 3, 5}
	for _, attempt := range attempts {
		var seen = make(map[time.Duration]bool)
		for i := 0; i < 100; i++ {
			d := backoff(attempt, min, max)
			if d < 0 {
				t.Errorf("backoff(%d) = %v, want >= 0", attempt, d)
			}
			if d > max {
				t.Errorf("backoff(%d) = %v, want <= %v", attempt, d, max)
			}
			seen[d] = true
		}
		// Jitter should produce variation
		if len(seen) < 2 {
			t.Errorf("backoff(%d) produced no jitter variation over 100 iterations", attempt)
		}
	}

	// Verify growth: attempt 0 should generally be around min, attempt 3 around min*8
	var sum0, sum3 time.Duration
	const n = 1000
	for i := 0; i < n; i++ {
		sum0 += backoff(0, min, max)
		sum3 += backoff(3, min, max)
	}
	avg0 := sum0 / time.Duration(n)
	avg3 := sum3 / time.Duration(n)
	if avg3 <= avg0 {
		t.Errorf("expected backoff growth: avg attempt 0 = %v, avg attempt 3 = %v", avg0, avg3)
	}
}

func TestClient_DoRequest_RetryOnServerError(t *testing.T) {
	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		if n <= 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"code":"INTERNAL","message":"server error"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey,
		WithBaseURL(srv.URL),
		WithMaxRetries(3),
		WithRetryWaitMin(1*time.Millisecond),
		WithRetryWaitMax(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, "")
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	resp.Body.Close()

	if got := counter.Load(); got != 3 {
		t.Errorf("request count = %d, want 3", got)
	}
}

func TestClient_DoRequest_RetryExhausted(t *testing.T) {
	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":"INTERNAL","message":"server error"}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey,
		WithBaseURL(srv.URL),
		WithMaxRetries(2),
		WithRetryWaitMin(1*time.Millisecond),
		WithRetryWaitMax(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.doRequest(context.Background(), http.MethodGet, "/test", nil, "")
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}

	var intErr *InternalError
	if !errors.As(err, &intErr) {
		t.Fatalf("error type = %T, want *InternalError", err)
	}

	if got := counter.Load(); got != 3 {
		t.Errorf("request count = %d, want 3 (1 initial + 2 retries)", got)
	}
}

func TestClient_DoRequest_NoRetryOn4xx(t *testing.T) {
	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"BAD_REQUEST","message":"bad request"}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey,
		WithBaseURL(srv.URL),
		WithMaxRetries(3),
		WithRetryWaitMin(1*time.Millisecond),
		WithRetryWaitMax(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.doRequest(context.Background(), http.MethodGet, "/test", nil, "")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	if got := counter.Load(); got != 1 {
		t.Errorf("request count = %d, want 1 (no retries for 4xx)", got)
	}
}
