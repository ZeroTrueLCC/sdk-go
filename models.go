package zerotrue

// ContentType represents the type of content being analyzed.
type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
	ContentTypeVideo ContentType = "video"
	ContentTypeCode  ContentType = "code"
	ContentTypeVoice ContentType = "voice"
	ContentTypeMusic ContentType = "music"
)

// Status represents the processing status of an analysis request.
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// AnalysisResult is the main result object returned by the API.
type AnalysisResult struct {
	AIProbability       float64          `json:"ai_probability"`
	HumanProbability    float64          `json:"human_probability"`
	CombinedProbability float64          `json:"combined_probability"`
	ResultType          string           `json:"result_type"`
	MLModel             string           `json:"ml_model"`
	MLModelVersion      *string          `json:"ml_model_version,omitempty"`
	Details             map[string]any   `json:"details,omitempty"`
	Feedback            *string          `json:"feedback,omitempty"`
	CreatedAt           *string          `json:"created_at,omitempty"`
	Status              *string          `json:"status,omitempty"`
	FileURL             *string          `json:"file_url,omitempty"`
	OriginalFilename    *string          `json:"original_filename,omitempty"`
	SizeBytes           *int64           `json:"size_bytes,omitempty"`
	SizeMB              *float64         `json:"size_mb,omitempty"`
	Resolution          *string          `json:"resolution,omitempty"`
	Length              *int             `json:"length,omitempty"`
	Content             *string          `json:"content,omitempty"`
	IsPrivateScan       *bool            `json:"is_private_scan,omitempty"`
	IsDeepScan          *bool            `json:"is_deep_scan,omitempty"`
	Price               *int             `json:"price,omitempty"`
	InferenceTimeMs     *int             `json:"inference_time_ms,omitempty"`
	APISchemaVersion    *string          `json:"api_schema_version,omitempty"`
	MetaMime            *string          `json:"meta_mime,omitempty"`
	MetaFileSizeBytes   *int64           `json:"meta_file_size_bytes,omitempty"`
	MetaSHA256          *string          `json:"meta_sha256,omitempty"`
	MetaContentURL      *string          `json:"meta_content_url,omitempty"`
	MetaContentType     *string          `json:"meta_content_type,omitempty"`
	DetailsSummary      *DetailsSummary  `json:"details_summary,omitempty"`
	DetailsExtra        map[string]any   `json:"details_extra,omitempty"`
	SuspectedModels     []SuspectedModel `json:"suspected_models,omitempty"`
	Segments            []Segment        `json:"segments,omitempty"`
	PreviewURL          *string          `json:"preview_url,omitempty"`
	ViewsCount          *int             `json:"views_count,omitempty"`
}

// SuspectedModel represents a suspected AI model with its confidence.
type SuspectedModel struct {
	ModelName     string  `json:"model_name"`
	ConfidencePct float64 `json:"confidence_pct"`
}

// Segment represents a segment of analyzed content.
type Segment struct {
	Label         string   `json:"label"`
	ConfidencePct float64  `json:"confidence_pct"`
	StartChar     *int     `json:"start_char,omitempty"`
	EndChar       *int     `json:"end_char,omitempty"`
	StartLine     *int     `json:"start_line,omitempty"`
	EndLine       *int     `json:"end_line,omitempty"`
	StartS        *float64 `json:"start_s,omitempty"`
	EndS          *float64 `json:"end_s,omitempty"`
	Timecode      *string  `json:"timecode,omitempty"`
}

// DetailsSummary provides a summary of the analysis details.
type DetailsSummary struct {
	OverallAssessment string  `json:"overall_assessment"`
	ProcessingTimeS   float64 `json:"processing_time_s"`
	GenTechnique      string  `json:"gen_technique"`
}

// AnalyzeOptions configures an analysis request.
type AnalyzeOptions struct {
	IsDeepScan    bool `json:"is_deep_scan"`
	IsPrivateScan bool `json:"is_private_scan"`
}

// CheckInput represents the input for a check request.
type CheckInput struct {
	Type     string `json:"type"`
	Value    string `json:"value,omitempty"`
	FilePath string `json:"-"`
}

// CheckOptions configures a check request.
type CheckOptions struct {
	IsDeepScan     bool           `json:"is_deep_scan"`
	IsPrivateScan  bool           `json:"is_private_scan"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// CheckResponse is the response from submitting a check.
type CheckResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// CheckResult is the result of a completed check.
type CheckResult struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Result *AnalysisResult `json:"result,omitempty"`
}

// APIInfo describes the API endpoint information.
type APIInfo struct {
	Name             string              `json:"name"`
	Version          string              `json:"version"`
	Description      string              `json:"description"`
	Endpoints        map[string]string   `json:"endpoints"`
	SupportedFormats map[string][]string `json:"supported_formats"`
}
