package zerotrue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type resultResponse struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Result *AnalysisResult `json:"result"`
}

// GetResult retrieves a previously computed analysis result by content ID.
// The api_key is sent as a query parameter.
func (c *Client) GetResult(ctx context.Context, contentID string) (*AnalysisResult, error) {
	if contentID == "" {
		return nil, fmt.Errorf("zerotrue: content ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/result/%s?api_key=%s", contentID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseErrorResponse(resp)
	}
	defer resp.Body.Close()

	var result resultResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("zerotrue: failed to decode response: %w", err)
	}

	if result.Result == nil {
		return nil, fmt.Errorf("zerotrue: empty result in response")
	}

	return result.Result, nil
}
