package unifi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var ErrNotFound = errors.New("not found")

// maxErrorBodySize caps how much of an error response body we buffer before
// decoding, so a hostile or runaway error page (e.g. a multi-megabyte HTML
// gateway response) can never exhaust memory while we build a ServerError.
const maxErrorBodySize = 1 << 20 // 1 MiB

type Meta struct {
	RC      string `json:"rc"`
	Message string `json:"msg"`
}

func (m *Meta) error() error {
	if m.RC != "ok" {
		return &ServerError{
			ErrorCode: m.RC,
			Message:   m.Message,
		}
	}

	return nil
}

type DefaultResponseErrorHandler struct{}

type apiV2ResponseError struct {
	Code      string                    `json:"code"`
	ErrorCode int                       `json:"errorCode"`
	Message   string                    `json:"message"`
	Details   apiV2ResponseErrorDetails `json:"details"`
}

type apiV2ResponseErrorDetails struct {
	// probably there are more fields, but I didn't get any response with more fields
	InvalidFields []string `json:"invalid_fields"`
}

type apiV1ResponseError struct {
	Meta Meta                     `json:"Meta"`
	Data []apiV1ResponseErrorData `json:"data"`
}

type apiV1ResponseErrorData struct {
	Meta            Meta                  `json:"Meta"`
	ValidationError ServerValidationError `json:"validationError"`
	RC              string                `json:"rc"`
	Message         string                `json:"msg"`
}

type apiResponseError struct {
	apiV1ResponseError
	apiV2ResponseError
}

type ServerValidationError struct {
	Field   string `json:"field"`
	Pattern string `json:"pattern"`
}

type ServerErrorDetails struct {
	Message         string
	ValidationError ServerValidationError
}

type ServerError struct {
	StatusCode    int
	RequestMethod string
	RequestURL    string
	Message       string
	ErrorCode     string
	Details       []ServerErrorDetails
}

func (s *ServerError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Server error (%d) for %s %s: %s", s.StatusCode, s.RequestMethod, s.RequestURL, s.Message)
	for _, detail := range s.Details {
		b.WriteString("\n")
		if detail.Message != "" {
			b.WriteString(detail.Message + ": ")
		}
		if detail.ValidationError.Field != "" && detail.ValidationError.Pattern != "" {
			fmt.Fprintf(&b, "field '%s' should match '%s'", detail.ValidationError.Field, detail.ValidationError.Pattern)
		} else if detail.ValidationError.Field != "" {
			fmt.Fprintf(&b, "field '%s' is invalid", detail.ValidationError.Field)
		} else if detail.ValidationError.Pattern != "" {
			fmt.Fprintf(&b, "field should match '%s'", detail.ValidationError.Pattern)
		}
	}
	return b.String()
}

// Is lets a *ServerError participate in errors.Is. A real HTTP 404 maps to the
// ErrNotFound sentinel so that errors.Is(err, ErrNotFound) holds uniformly for
// both a genuine 404 response and the existing empty-data 200 case.
func (s *ServerError) Is(target error) bool {
	if target == ErrNotFound {
		return s.StatusCode == http.StatusNotFound
	}
	return false
}

func parseApiV2Error(err apiV2ResponseError, serverError *ServerError) {
	serverError.Message = err.Message
	serverError.ErrorCode = err.Code
	for _, field := range err.Details.InvalidFields {
		details := ServerErrorDetails{}
		details.ValidationError.Field = field
		serverError.Details = append(serverError.Details, details)
	}
}

func parseApiV1Error(err apiV1ResponseError, serverError *ServerError) {
	for _, d := range err.Data {
		if d.Meta.RC == "error" || d.RC == "error" {
			details := ServerErrorDetails{}
			details.Message = d.Message
			if details.Message == "" {
				details.Message = d.Meta.Message
			}
			if d.ValidationError.Field != "" || d.ValidationError.Pattern != "" {
				details.ValidationError = d.ValidationError
			}
			serverError.Details = append(serverError.Details, details)
		}
	}
	if serverError.Message == "" {
		serverError.Message = err.Meta.Message
	}
	serverError.ErrorCode = err.Meta.RC
}

func (d *DefaultResponseErrorHandler) HandleError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	serverError := ServerError{
		StatusCode:    resp.StatusCode,
		RequestMethod: resp.Request.Method,
		RequestURL:    resp.Request.URL.String(),
	}

	// Read the body ONCE into a capped buffer before attempting to decode, so the
	// raw text remains available for a fallback message. On an empty body (common
	// for 401/403) or a non-JSON body (e.g. an HTML 502/504 gateway page) we still
	// return a fully-populated *ServerError carrying the status/method/URL rather
	// than leaking a bare io.EOF or "invalid character" decode error.
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if readErr != nil {
		serverError.Message = fmt.Sprintf("unable to read error response body: %v", readErr)
		return &serverError
	}

	var errBody apiResponseError
	if decodeErr := json.NewDecoder(bytes.NewReader(body)).Decode(&errBody); decodeErr != nil {
		// Non-JSON or empty body: surface a useful message instead of the raw
		// decode error so the status code and request context are never lost.
		serverError.Message = errorBodyFallbackMessage(body, decodeErr)
		return &serverError
	}

	if errBody.Code != "" || errBody.Message != "" {
		parseApiV2Error(errBody.apiV2ResponseError, &serverError)
	} else {
		parseApiV1Error(errBody.apiV1ResponseError, &serverError)
	}

	return &serverError
}

// errorBodyFallbackMessage builds a human-readable Message for a ServerError when
// the response body could not be decoded as a UniFi error envelope. It prefers
// the raw body text (trimmed) and falls back to the decode error when the body is
// empty.
func errorBodyFallbackMessage(body []byte, decodeErr error) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Sprintf("empty error response body: %v", decodeErr)
	}
	return trimmed
}
