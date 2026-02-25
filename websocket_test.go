package zerotrue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsTestServer creates an httptest.Server that upgrades connections to WebSocket
// and invokes handler with the resulting connection.
func wsTestServer(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade failed: %v", err)
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestClient_WaitForResult_Success(t *testing.T) {
	server := wsTestServer(t, func(conn *websocket.Conn) {
		msg := map[string]any{
			"ai_probability":       0.92,
			"human_probability":    0.08,
			"combined_probability": 0.92,
			"result_type":          "image_analysis",
			"ml_model":             "model-v2",
			"status":               "completed",
		}
		data, _ := json.Marshal(msg)
		_ = conn.WriteMessage(websocket.TextMessage, data)
	})

	client, err := NewClient("zt_testkey", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.WaitForResult(context.Background(), "test-item-123")
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}

	if result.AIProbability != 0.92 {
		t.Errorf("AIProbability = %v, want 0.92", result.AIProbability)
	}
	if result.HumanProbability != 0.08 {
		t.Errorf("HumanProbability = %v, want 0.08", result.HumanProbability)
	}
	if result.CombinedProbability != 0.92 {
		t.Errorf("CombinedProbability = %v, want 0.92", result.CombinedProbability)
	}
	if result.ResultType != "image_analysis" {
		t.Errorf("ResultType = %v, want image_analysis", result.ResultType)
	}
	if result.MLModel != "model-v2" {
		t.Errorf("MLModel = %v, want model-v2", result.MLModel)
	}
	if result.Status == nil || *result.Status != "completed" {
		t.Errorf("Status = %v, want completed", result.Status)
	}
}

func TestClient_WaitForResult_FieldRemapping(t *testing.T) {
	server := wsTestServer(t, func(conn *websocket.Conn) {
		msg := map[string]any{
			"id":                             "uuid-123",
			"ai_probability":                 0.85,
			"human_probability":              0.15,
			"combined_probability":           0.85,
			"result_type":                    "text_analysis",
			"ml_model":                       "model1",
			"content_item_status":            "completed",
			"content_item_url":               "https://cdn.example.com/file.png",
			"content_item_is_private_scan":   true,
			"content_item_is_deep_scan":      false,
			"content_item_price":             1,
		}
		data, _ := json.Marshal(msg)
		_ = conn.WriteMessage(websocket.TextMessage, data)
	})

	client, err := NewClient("zt_testkey", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.WaitForResult(context.Background(), "item-456")
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}

	if result.Status == nil || *result.Status != "completed" {
		t.Errorf("Status = %v, want completed", result.Status)
	}
	if result.FileURL == nil || *result.FileURL != "https://cdn.example.com/file.png" {
		t.Errorf("FileURL = %v, want https://cdn.example.com/file.png", result.FileURL)
	}
	if result.IsPrivateScan == nil || *result.IsPrivateScan != true {
		t.Errorf("IsPrivateScan = %v, want true", result.IsPrivateScan)
	}
	if result.IsDeepScan == nil || *result.IsDeepScan != false {
		t.Errorf("IsDeepScan = %v, want false", result.IsDeepScan)
	}
	if result.AIProbability != 0.85 {
		t.Errorf("AIProbability = %v, want 0.85", result.AIProbability)
	}
}

func TestClient_WaitForResult_ContextCancel(t *testing.T) {
	server := wsTestServer(t, func(conn *websocket.Conn) {
		// Block without sending anything.
		time.Sleep(10 * time.Second)
	})

	client, err := NewClient("zt_testkey", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.WaitForResult(ctx, "item-timeout")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ctx.Err() == nil {
		t.Fatal("expected context to be done")
	}
}

func TestClient_WaitForResult_EmptyID(t *testing.T) {
	client, err := NewClient("zt_testkey")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.WaitForResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
	want := "zerotrue: content item ID cannot be empty"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestHTTPToWSURL(t *testing.T) {
	tests := []struct {
		baseURL string
		path    string
		want    string
	}{
		{"https://api.example.com", "/ws/test", "wss://api.example.com/ws/test"},
		{"http://localhost:8080", "/ws/test", "ws://localhost:8080/ws/test"},
	}

	for _, tt := range tests {
		got := httpToWSURL(tt.baseURL, tt.path)
		if got != tt.want {
			t.Errorf("httpToWSURL(%q, %q) = %q, want %q", tt.baseURL, tt.path, got, tt.want)
		}
	}
}

func TestRemapWSFields(t *testing.T) {
	input := map[string]any{
		"content_item_status":          "completed",
		"content_item_url":             "https://example.com/file.png",
		"content_item_is_private_scan": true,
		"content_item_price":           2,
		"ai_probability":               0.9,
	}

	result := remapWSFields(input)

	if result["status"] != "completed" {
		t.Errorf("status = %v, want completed", result["status"])
	}
	if result["file_url"] != "https://example.com/file.png" {
		t.Errorf("file_url = %v, want https://example.com/file.png", result["file_url"])
	}
	if result["is_private_scan"] != true {
		t.Errorf("is_private_scan = %v, want true", result["is_private_scan"])
	}
	if result["price"] != 2 {
		t.Errorf("price = %v, want 2", result["price"])
	}
	if result["ai_probability"] != 0.9 {
		t.Errorf("ai_probability = %v, want 0.9", result["ai_probability"])
	}

	// Originals must be deleted.
	for oldKey := range wsFieldMapping {
		if _, ok := result[oldKey]; ok {
			t.Errorf("old key %q should have been deleted", oldKey)
		}
	}
}
