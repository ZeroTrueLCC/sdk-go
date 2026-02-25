package zerotrue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type checkRequest struct {
	Input          CheckInput     `json:"input"`
	IsDeepScan     bool           `json:"is_deep_scan"`
	IsPrivateScan  bool           `json:"is_private_scan"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// CreateCheck submits content for AI detection analysis.
func (c *Client) CreateCheck(ctx context.Context, input CheckInput, opts *CheckOptions) (*CheckResponse, error) {
	if opts == nil {
		opts = &CheckOptions{IsPrivateScan: true}
	}

	var resp *http.Response
	var err error

	if input.Type == "file" {
		resp, err = c.createCheckFile(ctx, input, opts)
	} else {
		resp, err = c.createCheckJSON(ctx, input, opts)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zerotrue: unexpected status %d", resp.StatusCode)
	}

	var cr CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("zerotrue: failed to decode check response: %w", err)
	}
	return &cr, nil
}

func (c *Client) createCheckJSON(ctx context.Context, input CheckInput, opts *CheckOptions) (*http.Response, error) {
	req := checkRequest{
		Input:          input,
		IsDeepScan:     opts.IsDeepScan,
		IsPrivateScan:  opts.IsPrivateScan,
		IdempotencyKey: opts.IdempotencyKey,
		Metadata:       opts.Metadata,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: failed to marshal check request: %w", err)
	}
	return c.doRequest(ctx, "POST", "/api/v1/check", bytes.NewReader(body), "application/json")
}

func (c *Client) createCheckFile(ctx context.Context, input CheckInput, opts *CheckOptions) (*http.Response, error) {
	f, err := os.Open(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("zerotrue: failed to open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_type", "file"); err != nil {
		return nil, err
	}
	if err := w.WriteField("is_deep_scan", fmt.Sprintf("%t", opts.IsDeepScan)); err != nil {
		return nil, err
	}
	if err := w.WriteField("is_private_scan", fmt.Sprintf("%t", opts.IsPrivateScan)); err != nil {
		return nil, err
	}

	part, err := w.CreateFormFile("input_file", filepath.Base(input.FilePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return c.doRequest(ctx, "POST", "/api/v1/check", &buf, w.FormDataContentType())
}

// GetCheck retrieves the status and result of a check by its ID.
func (c *Client) GetCheck(ctx context.Context, checkID string) (*CheckResult, error) {
	if checkID == "" {
		return nil, fmt.Errorf("zerotrue: check ID must not be empty")
	}

	path := fmt.Sprintf("/api/v1/checks/%s/", checkID)
	resp, err := c.doRequest(ctx, "GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("zerotrue: failed to decode check result: %w", err)
	}
	return &result, nil
}
