package zerotrue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const analyzeSuccessJSON = `{
	"id": "test-uuid",
	"status": "completed",
	"result": {
		"ai_probability": 0.85,
		"human_probability": 0.15,
		"combined_probability": 0.85,
		"result_type": "text_analysis",
		"ml_model": "test-model"
	}
}`

func analyzeHandler(t *testing.T, wantField, wantValue string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("api_key"); got != testAPIKey {
			t.Errorf("api_key = %q, want %q", got, testAPIKey)
		}
		if wantField != "" {
			if got := r.FormValue(wantField); wantValue != "" && got != wantValue {
				t.Errorf("%s = %q, want %q", wantField, got, wantValue)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(analyzeSuccessJSON))
	}
}

func assertAnalysisResult(t *testing.T, res *AnalysisResult) {
	t.Helper()
	if res.AIProbability != 0.85 {
		t.Errorf("AIProbability = %v, want 0.85", res.AIProbability)
	}
	if res.HumanProbability != 0.15 {
		t.Errorf("HumanProbability = %v, want 0.15", res.HumanProbability)
	}
	if res.CombinedProbability != 0.85 {
		t.Errorf("CombinedProbability = %v, want 0.85", res.CombinedProbability)
	}
	if res.ResultType != "text_analysis" {
		t.Errorf("ResultType = %q, want text_analysis", res.ResultType)
	}
	if res.MLModel != "test-model" {
		t.Errorf("MLModel = %q, want test-model", res.MLModel)
	}
}

func TestClient_AnalyzeFile_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "analyze-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("sample file content")
	tmpFile.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("api_key"); got != testAPIKey {
			t.Errorf("api_key = %q, want %q", got, testAPIKey)
		}
		if _, fh, err := r.FormFile("file"); err != nil {
			t.Errorf("FormFile: %v", err)
		} else if fh.Size == 0 {
			t.Error("uploaded file is empty")
		}
		if got := r.FormValue("is_deep_scan"); got != "false" {
			t.Errorf("is_deep_scan = %q, want false", got)
		}
		if got := r.FormValue("is_private_scan"); got != "true" {
			t.Errorf("is_private_scan = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(analyzeSuccessJSON))
	}))
	defer srv.Close()

	c, _ := NewClient(testAPIKey, WithBaseURL(srv.URL))
	res, err := c.AnalyzeFile(context.Background(), tmpFile.Name(), nil)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}
	assertAnalysisResult(t, res)
}

func TestClient_AnalyzeFile_FileNotFound(t *testing.T) {
	c, _ := NewClient(testAPIKey, WithBaseURL("http://localhost"))
	_, err := c.AnalyzeFile(context.Background(), "/nonexistent/path/file.txt", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error = %q, want 'no such file'", err.Error())
	}
}

func TestClient_AnalyzeText_Success(t *testing.T) {
	c, _ := newTestClient(t, analyzeHandler(t, "text", "hello world"))
	res, err := c.AnalyzeText(context.Background(), "hello world", nil)
	if err != nil {
		t.Fatalf("AnalyzeText error: %v", err)
	}
	assertAnalysisResult(t, res)
}

func TestClient_AnalyzeText_Empty(t *testing.T) {
	c, _ := NewClient(testAPIKey)
	_, err := c.AnalyzeText(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !strings.Contains(err.Error(), "text cannot be empty") {
		t.Errorf("error = %q, want 'text cannot be empty'", err.Error())
	}
}

func TestClient_AnalyzeURL_Success(t *testing.T) {
	c, _ := newTestClient(t, analyzeHandler(t, "url", "https://example.com"))
	res, err := c.AnalyzeURL(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("AnalyzeURL error: %v", err)
	}
	assertAnalysisResult(t, res)
}

func TestClient_AnalyzeURL_Empty(t *testing.T) {
	c, _ := NewClient(testAPIKey)
	_, err := c.AnalyzeURL(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty url")
	}
	if !strings.Contains(err.Error(), "url cannot be empty") {
		t.Errorf("error = %q, want 'url cannot be empty'", err.Error())
	}
}

func TestClient_Analyze_Options(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("is_deep_scan"); got != "true" {
			t.Errorf("is_deep_scan = %q, want true", got)
		}
		if got := r.FormValue("is_private_scan"); got != "false" {
			t.Errorf("is_private_scan = %q, want false", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(analyzeSuccessJSON))
	}))
	defer srv.Close()

	c, _ := NewClient(testAPIKey, WithBaseURL(srv.URL))
	opts := &AnalyzeOptions{IsDeepScan: true, IsPrivateScan: false}
	res, err := c.AnalyzeText(context.Background(), "test text", opts)
	if err != nil {
		t.Fatalf("AnalyzeText error: %v", err)
	}
	assertAnalysisResult(t, res)
}

func TestClient_Analyze_DefaultOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("is_deep_scan"); got != "false" {
			t.Errorf("is_deep_scan = %q, want false", got)
		}
		if got := r.FormValue("is_private_scan"); got != "true" {
			t.Errorf("is_private_scan = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(analyzeSuccessJSON))
	}))
	defer srv.Close()

	c, _ := NewClient(testAPIKey, WithBaseURL(srv.URL))
	res, err := c.AnalyzeText(context.Background(), "test text", nil)
	if err != nil {
		t.Fatalf("AnalyzeText error: %v", err)
	}
	assertAnalysisResult(t, res)
}

func TestClient_Analyze_NoAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Authorization header = %q, want empty (api_key should be in form data)", auth)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("api_key"); got != testAPIKey {
			t.Errorf("api_key form field = %q, want %q", got, testAPIKey)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(analyzeSuccessJSON))
	}))
	defer srv.Close()

	c, _ := NewClient(testAPIKey, WithBaseURL(srv.URL))
	_, err := c.AnalyzeText(context.Background(), "check no auth", nil)
	if err != nil {
		t.Fatalf("AnalyzeText error: %v", err)
	}
}
