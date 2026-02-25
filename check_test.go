package zerotrue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_CreateCheck_Text(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/check" {
			t.Errorf("path = %s, want /api/v1/check", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer zt_testkey1234567890abcdef12345678" {
			t.Errorf("Authorization = %q", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body checkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Input.Type != "text" {
			t.Errorf("input.type = %q, want text", body.Input.Type)
		}
		if body.Input.Value != "test text" {
			t.Errorf("input.value = %q, want %q", body.Input.Value, "test text")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"uuid-123","status":"queued"}`))
	})

	resp, err := c.CreateCheck(context.Background(), CheckInput{Type: "text", Value: "test text"}, nil)
	if err != nil {
		t.Fatalf("CreateCheck error: %v", err)
	}
	if resp.ID != "uuid-123" {
		t.Errorf("ID = %q, want uuid-123", resp.ID)
	}
	if resp.Status != "queued" {
		t.Errorf("Status = %q, want queued", resp.Status)
	}
}

func TestClient_CreateCheck_URL(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body checkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Input.Type != "url" {
			t.Errorf("input.type = %q, want url", body.Input.Type)
		}
		if body.Input.Value != "https://example.com" {
			t.Errorf("input.value = %q, want https://example.com", body.Input.Value)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"uuid-456","status":"queued"}`))
	})

	resp, err := c.CreateCheck(context.Background(), CheckInput{Type: "url", Value: "https://example.com"}, nil)
	if err != nil {
		t.Fatalf("CreateCheck error: %v", err)
	}
	if resp.ID != "uuid-456" {
		t.Errorf("ID = %q, want uuid-456", resp.ID)
	}
}

func TestClient_CreateCheck_File(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(fpath, []byte("file content"), 0644); err != nil {
		t.Fatal(err)
	}

	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer zt_testkey1234567890abcdef12345678" {
			t.Errorf("Authorization = %q", auth)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if v := r.FormValue("input_type"); v != "file" {
			t.Errorf("input_type = %q, want file", v)
		}
		f, _, err := r.FormFile("input_file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if string(data) != "file content" {
			t.Errorf("file content = %q", string(data))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"uuid-789","status":"queued"}`))
	})

	resp, err := c.CreateCheck(context.Background(), CheckInput{Type: "file", FilePath: fpath}, nil)
	if err != nil {
		t.Fatalf("CreateCheck error: %v", err)
	}
	if resp.ID != "uuid-789" {
		t.Errorf("ID = %q, want uuid-789", resp.ID)
	}
}

func TestClient_CreateCheck_WithIdempotencyKey(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body checkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.IdempotencyKey != "my-key-123" {
			t.Errorf("idempotency_key = %q, want my-key-123", body.IdempotencyKey)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"uuid-idem","status":"queued"}`))
	})

	opts := &CheckOptions{IdempotencyKey: "my-key-123"}
	resp, err := c.CreateCheck(context.Background(), CheckInput{Type: "text", Value: "test"}, opts)
	if err != nil {
		t.Fatalf("CreateCheck error: %v", err)
	}
	if resp.ID != "uuid-idem" {
		t.Errorf("ID = %q, want uuid-idem", resp.ID)
	}
}

func TestClient_CreateCheck_DefaultOptions(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body checkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.IsDeepScan != false {
			t.Errorf("is_deep_scan = %v, want false", body.IsDeepScan)
		}
		if body.IsPrivateScan != true {
			t.Errorf("is_private_scan = %v, want true", body.IsPrivateScan)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"uuid-def","status":"queued"}`))
	})

	resp, err := c.CreateCheck(context.Background(), CheckInput{Type: "text", Value: "defaults"}, nil)
	if err != nil {
		t.Fatalf("CreateCheck error: %v", err)
	}
	if resp.ID != "uuid-def" {
		t.Errorf("ID = %q, want uuid-def", resp.ID)
	}
}

func TestClient_GetCheck_Completed(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "uuid-completed",
			"status": "completed",
			"result": {
				"ai_probability": 0.95,
				"human_probability": 0.05,
				"combined_probability": 0.95,
				"result_type": "text",
				"ml_model": "model-v1",
				"created_at": "2025-01-01T00:00:00Z"
			}
		}`))
	})

	result, err := c.GetCheck(context.Background(), "uuid-completed")
	if err != nil {
		t.Fatalf("GetCheck error: %v", err)
	}
	if result.ID != "uuid-completed" {
		t.Errorf("ID = %q, want uuid-completed", result.ID)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if result.Result == nil {
		t.Fatal("Result is nil")
	}
	if result.Result.AIProbability != 0.95 {
		t.Errorf("AIProbability = %v, want 0.95", result.Result.AIProbability)
	}
	if result.Result.HumanProbability != 0.05 {
		t.Errorf("HumanProbability = %v, want 0.05", result.Result.HumanProbability)
	}
	if result.Result.CombinedProbability != 0.95 {
		t.Errorf("CombinedProbability = %v, want 0.95", result.Result.CombinedProbability)
	}
	if result.Result.ResultType != "text" {
		t.Errorf("ResultType = %q, want text", result.Result.ResultType)
	}
	if result.Result.MLModel != "model-v1" {
		t.Errorf("MLModel = %q, want model-v1", result.Result.MLModel)
	}
}

func TestClient_GetCheck_Processing(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"uuid-proc","status":"processing","result":{"status":"processing","created_at":"2025-01-01T00:00:00Z","ai_probability":0,"human_probability":0,"combined_probability":0,"result_type":"","ml_model":""}}`))
	})

	result, err := c.GetCheck(context.Background(), "uuid-proc")
	if err != nil {
		t.Fatalf("GetCheck error: %v", err)
	}
	if result.Status != "processing" {
		t.Errorf("Status = %q, want processing", result.Status)
	}
}

func TestClient_GetCheck_TrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/") {
			t.Errorf("path = %q, want trailing slash", r.URL.Path)
		}
		if r.URL.Path != "/api/v1/checks/check-id-1/" {
			t.Errorf("path = %q, want /api/v1/checks/check-id-1/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"check-id-1","status":"completed"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("zt_testkey1234567890abcdef12345678", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := c.GetCheck(context.Background(), "check-id-1")
	if err != nil {
		t.Fatalf("GetCheck error: %v", err)
	}
	if result.ID != "check-id-1" {
		t.Errorf("ID = %q, want check-id-1", result.ID)
	}
}

func TestClient_GetCheck_EmptyID(t *testing.T) {
	c, err := NewClient("zt_testkey1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetCheck(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty check ID")
	}
}
