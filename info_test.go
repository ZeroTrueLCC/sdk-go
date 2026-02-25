package zerotrue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("zt_testkey1234567890abcdef12345678", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestClient_GetInfo_Success(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/info" {
			t.Errorf("path = %s, want /api/v1/info", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"name": "ZeroTrue API",
			"version": "1.2.3",
			"description": "AI content detection API",
			"endpoints": {
				"analyze": "/api/v1/analyze",
				"check": "/api/v1/check"
			},
			"supported_formats": {
				"text": ["txt", "md"],
				"image": ["png", "jpg"]
			}
		}`))
	})

	info, err := c.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetInfo error: %v", err)
	}

	if info.Name != "ZeroTrue API" {
		t.Errorf("Name = %q, want %q", info.Name, "ZeroTrue API")
	}
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", info.Version, "1.2.3")
	}
	if info.Description != "AI content detection API" {
		t.Errorf("Description = %q, want %q", info.Description, "AI content detection API")
	}
	if len(info.Endpoints) != 2 {
		t.Fatalf("Endpoints len = %d, want 2", len(info.Endpoints))
	}
	if info.Endpoints["analyze"] != "/api/v1/analyze" {
		t.Errorf("Endpoints[analyze] = %q", info.Endpoints["analyze"])
	}
	if info.Endpoints["check"] != "/api/v1/check" {
		t.Errorf("Endpoints[check] = %q", info.Endpoints["check"])
	}
	if len(info.SupportedFormats) != 2 {
		t.Fatalf("SupportedFormats len = %d, want 2", len(info.SupportedFormats))
	}
	if len(info.SupportedFormats["text"]) != 2 || info.SupportedFormats["text"][0] != "txt" {
		t.Errorf("SupportedFormats[text] = %v", info.SupportedFormats["text"])
	}
	if len(info.SupportedFormats["image"]) != 2 || info.SupportedFormats["image"][0] != "png" {
		t.Errorf("SupportedFormats[image] = %v", info.SupportedFormats["image"])
	}
}

func TestClient_GetInfo_ServerError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"server error"},"request_id":"req_1"}`))
	})

	_, err := c.GetInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	var intErr *InternalError
	if !errors.As(err, &intErr) {
		t.Fatalf("error type = %T, want *InternalError", err)
	}
	if intErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", intErr.StatusCode)
	}
	if intErr.Code != "INTERNAL" {
		t.Errorf("Code = %q, want INTERNAL", intErr.Code)
	}
	if intErr.Message != "server error" {
		t.Errorf("Message = %q, want %q", intErr.Message, "server error")
	}
	if intErr.RequestID != "req_1" {
		t.Errorf("RequestID = %q, want req_1", intErr.RequestID)
	}
}
