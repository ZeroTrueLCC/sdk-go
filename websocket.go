package zerotrue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// wsFieldMapping maps WebSocket content_item_* prefixed fields to their
// corresponding AnalysisResult JSON field names.
var wsFieldMapping = map[string]string{
	"content_item_status":            "status",
	"content_item_url":               "file_url",
	"content_item_original_filename": "original_filename",
	"content_item_size_bytes":        "size_bytes",
	"content_item_size_mb":           "size_mb",
	"content_item_resolution":        "resolution",
	"content_item_length":            "length",
	"content_item_content":           "content",
	"content_item_is_private_scan":   "is_private_scan",
	"content_item_is_deep_scan":      "is_deep_scan",
	"content_item_price":             "price",
}

// WaitForResult connects to the classification WebSocket and blocks until the
// server sends the analysis result for the given content item.
func (c *Client) WaitForResult(ctx context.Context, contentItemID string) (*AnalysisResult, error) {
	if contentItemID == "" {
		return nil, fmt.Errorf("zerotrue: content item ID cannot be empty")
	}

	wsURL := httpToWSURL(c.baseURL, "/ws/classification/"+contentItemID+"/")

	dialer := websocket.Dialer{}
	if t, ok := c.httpClient.Transport.(*http.Transport); ok {
		dialer.TLSClientConfig = t.TLSClientConfig
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: websocket dial: %w", err)
	}
	defer conn.Close()

	// Close the connection when the context is cancelled.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("zerotrue: websocket read: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(msg, &raw); err != nil {
		return nil, fmt.Errorf("zerotrue: websocket unmarshal: %w", err)
	}

	raw = remapWSFields(raw)

	remapped, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: websocket marshal: %w", err)
	}

	var result AnalysisResult
	if err := json.Unmarshal(remapped, &result); err != nil {
		return nil, fmt.Errorf("zerotrue: websocket unmarshal result: %w", err)
	}

	return &result, nil
}

// httpToWSURL converts an HTTP(S) base URL to the corresponding WS(S) URL
// and appends the given path.
func httpToWSURL(baseURL, path string) string {
	wsURL := baseURL
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	return wsURL + path
}

// remapWSFields renames content_item_* prefixed keys to their standard
// AnalysisResult field names.
func remapWSFields(data map[string]any) map[string]any {
	for oldKey, newKey := range wsFieldMapping {
		if val, ok := data[oldKey]; ok {
			data[newKey] = val
			delete(data, oldKey)
		}
	}
	return data
}
