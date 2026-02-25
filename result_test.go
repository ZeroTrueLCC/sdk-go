package zerotrue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetResult_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"uuid-123","status":"completed","data":{"ai_probability":0.9,"human_probability":0.1,"combined_probability":0.9,"result_type":"text_analysis","ml_model":"model1"}}`))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey, WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := c.GetResult(context.Background(), "content-abc")
	if err != nil {
		t.Fatalf("GetResult error: %v", err)
	}
	if result.AIProbability != 0.9 {
		t.Errorf("AIProbability = %v, want 0.9", result.AIProbability)
	}
	if result.MLModel != "model1" {
		t.Errorf("MLModel = %q, want %q", result.MLModel, "model1")
	}
}

func TestClient_GetResult_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"not found"},"request_id":"req_1"}`))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey, WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetResult(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error type = %T, want *NotFoundError", err)
	}
}

func TestClient_GetResult_APIKeyInQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey := r.URL.Query().Get("api_key")
		if gotKey != testAPIKey {
			t.Errorf("api_key query param = %q, want %q", gotKey, testAPIKey)
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Authorization header should be absent, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"uuid-123","status":"completed","data":{"ai_probability":0.5,"human_probability":0.5,"combined_probability":0.5,"result_type":"text_analysis","ml_model":"m"}}`))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(testAPIKey, WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetResult(context.Background(), "content-xyz")
	if err != nil {
		t.Fatalf("GetResult error: %v", err)
	}
}

func TestClient_GetResult_EmptyID(t *testing.T) {
	c, err := NewClient(testAPIKey)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty content ID")
	}
	if !strings.Contains(err.Error(), "content ID cannot be empty") {
		t.Errorf("error = %q, want mention of empty content ID", err.Error())
	}
}
