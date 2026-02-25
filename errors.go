package zerotrue

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError is the base error type for all API errors.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
}

func (e *APIError) Error() string {
	s := fmt.Sprintf("zerotrue: HTTP %d", e.StatusCode)
	if e.Code != "" {
		s += ": " + e.Code
	}
	s += " - " + e.Message
	if e.RequestID != "" {
		s += " (request_id: " + e.RequestID + ")"
	}
	return s
}

type AuthenticationError struct{ *APIError }
type ForbiddenError struct{ *APIError }
type RateLimitError struct{ *APIError }
type InsufficientCreditsError struct{ *APIError }
type ValidationError struct{ *APIError }
type NotFoundError struct{ *APIError }
type TimeoutError struct{ *APIError }
type InternalError struct{ *APIError }
type BadGatewayError struct{ *APIError }

func (e *AuthenticationError) Unwrap() error      { return e.APIError }
func (e *ForbiddenError) Unwrap() error           { return e.APIError }
func (e *RateLimitError) Unwrap() error           { return e.APIError }
func (e *InsufficientCreditsError) Unwrap() error { return e.APIError }
func (e *ValidationError) Unwrap() error          { return e.APIError }
func (e *NotFoundError) Unwrap() error            { return e.APIError }
func (e *TimeoutError) Unwrap() error             { return e.APIError }
func (e *InternalError) Unwrap() error            { return e.APIError }
func (e *BadGatewayError) Unwrap() error          { return e.APIError }

func newAPIError(statusCode int, code, message, requestID string) error {
	base := &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		RequestID:  requestID,
	}

	switch statusCode {
	case 401:
		return &AuthenticationError{base}
	case 403:
		return &ForbiddenError{base}
	case 404:
		return &NotFoundError{base}
	case 408:
		return &TimeoutError{base}
	case 422:
		return &ValidationError{base}
	case 429:
		return &RateLimitError{base}
	case 400:
		if code == "INSUFFICIENT_CREDITS" || code == "INSUFFICIENT_PAID_CREDITS" {
			return &InsufficientCreditsError{base}
		}
		return base
	case 500:
		return &InternalError{base}
	case 502:
		return &BadGatewayError{base}
	default:
		return base
	}
}

func parseErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return newAPIError(resp.StatusCode, "", http.StatusText(resp.StatusCode), "")
	}

	var parsed struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Error.Message == "" {
		return newAPIError(resp.StatusCode, "", http.StatusText(resp.StatusCode), "")
	}

	return newAPIError(resp.StatusCode, parsed.Error.Code, parsed.Error.Message, parsed.RequestID)
}
