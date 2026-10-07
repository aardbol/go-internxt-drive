package internxtclient

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError describes a non-2xx HTTP response from an Internxt API.
// It is returned (wrapped) by all service methods on API failures and
// can be inspected with errors.As or the helper predicates below.
type APIError struct {
	StatusCode int
	Method     string
	Endpoint   string
	Body       []byte
}

func (e *APIError) Error() string {
	const maxLen = 200
	bodyStr := string(e.Body)
	if len(bodyStr) > maxLen {
		bodyStr = bodyStr[:maxLen] + "...(truncated)"
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, bodyStr)
}

// AsAPIError extracts an *APIError from the error chain, if present.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsNotFound reports whether err is (or wraps) an APIError with status 404.
func IsNotFound(err error) bool {
	return hasStatus(err, http.StatusNotFound)
}

// IsRateLimited reports whether err is (or wraps) an APIError with status 429.
func IsRateLimited(err error) bool {
	return hasStatus(err, http.StatusTooManyRequests)
}

// IsPaymentRequired reports whether err is (or wraps) an APIError with status 402,
// which the drive API returns when a file size exceeds the user's plan.
func IsPaymentRequired(err error) bool {
	return hasStatus(err, http.StatusPaymentRequired)
}

// IsUnauthorized reports whether err is (or wraps) an APIError with status 401.
func IsUnauthorized(err error) bool {
	return hasStatus(err, http.StatusUnauthorized)
}

// IsServerError reports whether err is (or wraps) an APIError with a 5xx status.
func IsServerError(err error) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.StatusCode >= 500 && apiErr.StatusCode <= 599
}

func hasStatus(err error, code int) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.StatusCode == code
}
