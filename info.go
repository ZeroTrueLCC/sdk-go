package zerotrue

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetInfo returns API information. Does not require authentication.
func (c *Client) GetInfo(ctx context.Context) (*APIInfo, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/info", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info APIInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("zerotrue: failed to decode info response: %w", err)
	}
	return &info, nil
}
