package zerotrue

import (
	"encoding/json"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestAnalysisResult_JSON_RoundTrip(t *testing.T) {
	original := AnalysisResult{
		AIProbability:      0.85,
		HumanProbability:   0.15,
		CombinedProbability: 0.80,
		ResultType:         "ai_generated",
		MLModel:            "detector-v3",
		MLModelVersion:     ptr("3.1.0"),
		Details:            map[string]any{"key": "value"},
		Feedback:           ptr("looks AI"),
		CreatedAt:          ptr("2024-01-15T10:30:00Z"),
		Status:             ptr("completed"),
		FileURL:            ptr("https://example.com/file.png"),
		OriginalFilename:   ptr("file.png"),
		SizeBytes:          ptr(int64(1024)),
		SizeMB:             ptr(0.001),
		Resolution:         ptr("1920x1080"),
		Length:             ptr(120),
		Content:            ptr("hello world"),
		IsPrivateScan:      ptr(true),
		IsDeepScan:         ptr(false),
		Price:              ptr(10),
		InferenceTimeMs:    ptr(250),
		APISchemaVersion:   ptr("1.0"),
		MetaMime:           ptr("image/png"),
		MetaFileSizeBytes:  ptr(int64(2048)),
		MetaSHA256:         ptr("abc123"),
		MetaContentURL:     ptr("https://example.com/content"),
		MetaContentType:    ptr("image"),
		DetailsSummary: &DetailsSummary{
			OverallAssessment: "likely AI",
			ProcessingTimeS:   1.5,
			GenTechnique:      "diffusion",
		},
		DetailsExtra:    map[string]any{"extra": "data"},
		SuspectedModels: []SuspectedModel{{ModelName: "DALL-E", ConfidencePct: 75.5}},
		Segments:        []Segment{{Label: "ai", ConfidencePct: 90.0, StartChar: ptr(0), EndChar: ptr(100)}},
		PreviewURL:      ptr("https://example.com/preview.png"),
		ViewsCount:      ptr(42),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded AnalysisResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.AIProbability != original.AIProbability {
		t.Errorf("AIProbability: got %v, want %v", decoded.AIProbability, original.AIProbability)
	}
	if decoded.HumanProbability != original.HumanProbability {
		t.Errorf("HumanProbability: got %v, want %v", decoded.HumanProbability, original.HumanProbability)
	}
	if decoded.CombinedProbability != original.CombinedProbability {
		t.Errorf("CombinedProbability: got %v, want %v", decoded.CombinedProbability, original.CombinedProbability)
	}
	if decoded.ResultType != original.ResultType {
		t.Errorf("ResultType: got %v, want %v", decoded.ResultType, original.ResultType)
	}
	if decoded.MLModel != original.MLModel {
		t.Errorf("MLModel: got %v, want %v", decoded.MLModel, original.MLModel)
	}
	if decoded.MLModelVersion == nil || *decoded.MLModelVersion != *original.MLModelVersion {
		t.Errorf("MLModelVersion: got %v, want %v", decoded.MLModelVersion, original.MLModelVersion)
	}
	if decoded.DetailsSummary == nil {
		t.Fatal("DetailsSummary is nil")
	}
	if decoded.DetailsSummary.OverallAssessment != "likely AI" {
		t.Errorf("OverallAssessment: got %v", decoded.DetailsSummary.OverallAssessment)
	}
	if len(decoded.SuspectedModels) != 1 || decoded.SuspectedModels[0].ModelName != "DALL-E" {
		t.Errorf("SuspectedModels: got %v", decoded.SuspectedModels)
	}
	if len(decoded.Segments) != 1 || decoded.Segments[0].Label != "ai" {
		t.Errorf("Segments: got %v", decoded.Segments)
	}
	if decoded.ViewsCount == nil || *decoded.ViewsCount != 42 {
		t.Errorf("ViewsCount: got %v, want 42", decoded.ViewsCount)
	}
	if decoded.PreviewURL == nil || *decoded.PreviewURL != "https://example.com/preview.png" {
		t.Errorf("PreviewURL: got %v", decoded.PreviewURL)
	}

	// Verify snake_case JSON keys
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	expectedKeys := []string{
		"ai_probability", "human_probability", "combined_probability",
		"result_type", "ml_model", "ml_model_version", "details",
		"feedback", "created_at", "status", "file_url",
		"original_filename", "size_bytes", "size_mb", "resolution",
		"length", "content", "is_private_scan", "is_deep_scan",
		"price", "inference_time_ms", "api_schema_version",
		"meta_mime", "meta_file_size_bytes", "meta_sha256",
		"meta_content_url", "meta_content_type", "details_summary",
		"details_extra", "suspected_models", "segments",
		"preview_url", "views_count",
	}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}
}

func TestAnalysisResult_JSON_Nullable(t *testing.T) {
	minimal := AnalysisResult{
		AIProbability:      0.5,
		HumanProbability:   0.5,
		CombinedProbability: 0.5,
		ResultType:         "mixed",
		MLModel:            "detector-v1",
	}

	data, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	omittedKeys := []string{
		"ml_model_version", "details", "feedback", "created_at",
		"status", "file_url", "original_filename", "size_bytes",
		"size_mb", "resolution", "length", "content",
		"is_private_scan", "is_deep_scan", "price",
		"inference_time_ms", "api_schema_version", "meta_mime",
		"meta_file_size_bytes", "meta_sha256", "meta_content_url",
		"meta_content_type", "details_summary", "details_extra",
		"suspected_models", "segments", "preview_url", "views_count",
	}
	for _, key := range omittedKeys {
		if _, ok := raw[key]; ok {
			t.Errorf("key %q should be omitted when nil/empty", key)
		}
	}

	requiredKeys := []string{
		"ai_probability", "human_probability", "combined_probability",
		"result_type", "ml_model",
	}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("required key %q missing", key)
		}
	}
}

func TestSuspectedModel_JSON(t *testing.T) {
	tests := []struct {
		name  string
		model SuspectedModel
	}{
		{"basic", SuspectedModel{ModelName: "GPT-4", ConfidencePct: 92.3}},
		{"zero confidence", SuspectedModel{ModelName: "Unknown", ConfidencePct: 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.model)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded SuspectedModel
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if decoded.ModelName != tc.model.ModelName {
				t.Errorf("ModelName: got %q, want %q", decoded.ModelName, tc.model.ModelName)
			}
			if decoded.ConfidencePct != tc.model.ConfidencePct {
				t.Errorf("ConfidencePct: got %v, want %v", decoded.ConfidencePct, tc.model.ConfidencePct)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal map: %v", err)
			}
			if _, ok := raw["model_name"]; !ok {
				t.Error("missing JSON key model_name")
			}
			if _, ok := raw["confidence_pct"]; !ok {
				t.Error("missing JSON key confidence_pct")
			}
		})
	}
}

func TestSegment_JSON(t *testing.T) {
	tests := []struct {
		name    string
		segment Segment
	}{
		{
			"text segment",
			Segment{
				Label: "ai", ConfidencePct: 88.0,
				StartChar: ptr(0), EndChar: ptr(500),
				StartLine: ptr(1), EndLine: ptr(10),
			},
		},
		{
			"audio segment",
			Segment{
				Label: "human", ConfidencePct: 95.0,
				StartS: ptr(0.0), EndS: ptr(30.5),
				Timecode: ptr("00:00:00-00:00:30"),
			},
		},
		{
			"minimal segment",
			Segment{Label: "mixed", ConfidencePct: 50.0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.segment)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded Segment
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if decoded.Label != tc.segment.Label {
				t.Errorf("Label: got %q, want %q", decoded.Label, tc.segment.Label)
			}
			if decoded.ConfidencePct != tc.segment.ConfidencePct {
				t.Errorf("ConfidencePct: got %v, want %v", decoded.ConfidencePct, tc.segment.ConfidencePct)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal map: %v", err)
			}
			if _, ok := raw["label"]; !ok {
				t.Error("missing JSON key label")
			}
			if _, ok := raw["confidence_pct"]; !ok {
				t.Error("missing JSON key confidence_pct")
			}
		})
	}
}

func TestContentType_Values(t *testing.T) {
	tests := []struct {
		constant ContentType
		expected string
	}{
		{ContentTypeText, "text"},
		{ContentTypeImage, "image"},
		{ContentTypeVideo, "video"},
		{ContentTypeCode, "code"},
		{ContentTypeVoice, "voice"},
		{ContentTypeMusic, "music"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if string(tc.constant) != tc.expected {
				t.Errorf("got %q, want %q", tc.constant, tc.expected)
			}
		})
	}
}

func TestStatus_Values(t *testing.T) {
	tests := []struct {
		constant Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusProcessing, "processing"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if string(tc.constant) != tc.expected {
				t.Errorf("got %q, want %q", tc.constant, tc.expected)
			}
		})
	}
}
