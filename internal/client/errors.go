package client

import "fmt"

// APIError represents a non-OK response from the Middle Monitor API.
type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api %s %s: %d %s", e.Method, e.URL, e.StatusCode, e.Body)
}

// DecodeError represents a failure to decode a JSON response body.
type DecodeError struct {
	Cause error
	Body  string
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode: %s (body: %s)", e.Cause, e.Body)
}

func (e *DecodeError) Unwrap() error {
	return e.Cause
}
