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
	"time"
)

type analyzeResponse struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Result *AnalysisResult `json:"result"`
}

// AnalyzeFile uploads a file for AI-content analysis via the Gateway.
func (c *Client) AnalyzeFile(ctx context.Context, filePath string, opts *AnalyzeOptions) (*AnalysisResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("zerotrue: file path cannot be empty")
	}

	return c.doMultipartAnalyze(ctx, "/api/v1/analyze/file", func(w *multipart.Writer) error {
		if err := w.WriteField("api_key", c.apiKey); err != nil {
			return err
		}

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("zerotrue: failed to open file: %w", err)
		}
		defer f.Close()

		part, err := w.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, f); err != nil {
			return err
		}

		return writeAnalyzeOptions(w, opts)
	})
}

// AnalyzeText submits text for AI-content analysis via the Gateway.
func (c *Client) AnalyzeText(ctx context.Context, text string, opts *AnalyzeOptions) (*AnalysisResult, error) {
	if text == "" {
		return nil, fmt.Errorf("zerotrue: text cannot be empty")
	}

	return c.doMultipartAnalyze(ctx, "/api/v1/analyze/text", func(w *multipart.Writer) error {
		if err := w.WriteField("api_key", c.apiKey); err != nil {
			return err
		}
		if err := w.WriteField("text", text); err != nil {
			return err
		}
		return writeAnalyzeOptions(w, opts)
	})
}

// AnalyzeURL submits a URL for AI-content analysis via the Gateway.
func (c *Client) AnalyzeURL(ctx context.Context, rawURL string, opts *AnalyzeOptions) (*AnalysisResult, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("zerotrue: url cannot be empty")
	}

	return c.doMultipartAnalyze(ctx, "/api/v1/analyze/url", func(w *multipart.Writer) error {
		if err := w.WriteField("api_key", c.apiKey); err != nil {
			return err
		}
		if err := w.WriteField("url", rawURL); err != nil {
			return err
		}
		return writeAnalyzeOptions(w, opts)
	})
}

func writeAnalyzeOptions(w *multipart.Writer, opts *AnalyzeOptions) error {
	deep, private := "false", "true"
	if opts != nil {
		if opts.IsDeepScan {
			deep = "true"
		}
		if !opts.IsPrivateScan {
			private = "false"
		}
	}
	if err := w.WriteField("is_deep_scan", deep); err != nil {
		return err
	}
	return w.WriteField("is_private_scan", private)
}

func (c *Client) doMultipartAnalyze(ctx context.Context, path string, buildForm func(w *multipart.Writer) error) (*AnalysisResult, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := buildForm(w); err != nil {
		return nil, err
	}
	w.Close()

	bodyBytes := buf.Bytes()
	contentType := w.FormDataContentType()

	maxAttempts := c.maxRetries + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("zerotrue: failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("zerotrue: request failed: %w", err)
		}

		if resp.StatusCode >= 400 {
			if shouldRetry(resp.StatusCode) && attempt < maxAttempts-1 {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				wait := backoff(attempt, c.retryWaitMin, c.retryWaitMax)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
			return nil, parseErrorResponse(resp)
		}

		defer resp.Body.Close()

		var result analyzeResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("zerotrue: failed to decode response: %w", err)
		}

		if result.Result == nil {
			return nil, fmt.Errorf("zerotrue: empty result in response")
		}

		return result.Result, nil
	}

	return nil, fmt.Errorf("zerotrue: request failed after %d attempts", maxAttempts)
}
