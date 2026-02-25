package zerotrue

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{
		StatusCode: 400,
		Code:       "BAD_REQUEST",
		Message:    "invalid input",
		RequestID:  "req_123",
	}
	want := "zerotrue: HTTP 400: BAD_REQUEST - invalid input (request_id: req_123)"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAPIError_ErrorWithoutCode(t *testing.T) {
	e := &APIError{
		StatusCode: 500,
		Code:       "",
		Message:    "server error",
		RequestID:  "req_456",
	}
	want := "zerotrue: HTTP 500 - server error (request_id: req_456)"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAPIError_ErrorWithoutRequestID(t *testing.T) {
	e := &APIError{
		StatusCode: 400,
		Code:       "BAD_REQUEST",
		Message:    "invalid input",
		RequestID:  "",
	}
	want := "zerotrue: HTTP 400: BAD_REQUEST - invalid input"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewAPIError_Classification(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       string
		message    string
		requestID  string
		wantType   interface{}
		notType    interface{}
	}{
		{"401 auth", 401, "AUTHENTICATION_FAILED", "msg", "req_1", &AuthenticationError{}, nil},
		{"403 forbidden", 403, "FORBIDDEN", "msg", "req_2", &ForbiddenError{}, nil},
		{"404 not found", 404, "NOT_FOUND", "msg", "req_3", &NotFoundError{}, nil},
		{"408 timeout", 408, "TIMEOUT", "msg", "req_4", &TimeoutError{}, nil},
		{"422 validation", 422, "VALIDATION_ERROR", "msg", "req_5", &ValidationError{}, nil},
		{"429 rate limit", 429, "", "rate limit", "req_6", &RateLimitError{}, nil},
		{"400 insufficient credits", 400, "INSUFFICIENT_CREDITS", "msg", "req_7", &InsufficientCreditsError{}, nil},
		{"400 insufficient paid credits", 400, "INSUFFICIENT_PAID_CREDITS", "msg", "req_8", &InsufficientCreditsError{}, nil},
		{"400 bad request", 400, "BAD_REQUEST", "msg", "req_9", &APIError{}, &InsufficientCreditsError{}},
		{"500 internal", 500, "INTERNAL", "msg", "req_10", &InternalError{}, nil},
		{"502 bad gateway", 502, "BAD_GATEWAY", "msg", "req_11", &BadGatewayError{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newAPIError(tt.statusCode, tt.code, tt.message, tt.requestID)
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			switch target := tt.wantType.(type) {
			case *AuthenticationError:
				if !errors.As(err, &target) {
					t.Errorf("expected AuthenticationError, got %T", err)
				}
			case *ForbiddenError:
				if !errors.As(err, &target) {
					t.Errorf("expected ForbiddenError, got %T", err)
				}
			case *NotFoundError:
				if !errors.As(err, &target) {
					t.Errorf("expected NotFoundError, got %T", err)
				}
			case *TimeoutError:
				if !errors.As(err, &target) {
					t.Errorf("expected TimeoutError, got %T", err)
				}
			case *ValidationError:
				if !errors.As(err, &target) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			case *RateLimitError:
				if !errors.As(err, &target) {
					t.Errorf("expected RateLimitError, got %T", err)
				}
			case *InsufficientCreditsError:
				if !errors.As(err, &target) {
					t.Errorf("expected InsufficientCreditsError, got %T", err)
				}
			case *InternalError:
				if !errors.As(err, &target) {
					t.Errorf("expected InternalError, got %T", err)
				}
			case *BadGatewayError:
				if !errors.As(err, &target) {
					t.Errorf("expected BadGatewayError, got %T", err)
				}
			case *APIError:
				if !errors.As(err, &target) {
					t.Errorf("expected APIError, got %T", err)
				}
			}

			if tt.notType != nil {
				switch notTarget := tt.notType.(type) {
				case *InsufficientCreditsError:
					if errors.As(err, &notTarget) {
						t.Errorf("should not be InsufficientCreditsError, got %T", err)
					}
				}
			}
		})
	}
}

func TestParseErrorResponse(t *testing.T) {
	body := `{"error":{"code":"AUTHENTICATION_FAILED","message":"invalid token"},"request_id":"req_abc"}`
	resp := &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	err := parseErrorResponse(resp)
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthenticationError, got %T", err)
	}
	if authErr.Code != "AUTHENTICATION_FAILED" {
		t.Errorf("expected code AUTHENTICATION_FAILED, got %s", authErr.Code)
	}
	if authErr.Message != "invalid token" {
		t.Errorf("expected message 'invalid token', got %s", authErr.Message)
	}
	if authErr.RequestID != "req_abc" {
		t.Errorf("expected request_id req_abc, got %s", authErr.RequestID)
	}
}

func TestParseErrorResponse_InvalidJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("not json")),
	}

	err := parseErrorResponse(resp)
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Internal Server Error" {
		t.Errorf("expected message 'Internal Server Error', got %s", apiErr.Message)
	}
}
