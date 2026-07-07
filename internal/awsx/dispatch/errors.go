// SPDX-License-Identifier: MIT

package dispatch

import (
	"errors"

	"github.com/aws/smithy-go"
)

// APIError is a structured representation of an AWS API error, extracted
// generically from any service's error type via the smithy.APIError
// interface every AWS SDK v2 service implements — no per-service error
// handling is needed.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fault   string `json:"fault"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Code + ": " + e.Message
}

// mapError converts an AWS SDK error into an *APIError when it carries
// smithy.APIError information; other errors (e.g. network failures,
// context cancellation) are returned unchanged.
func mapError(err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return &APIError{
			Code:    apiErr.ErrorCode(),
			Message: apiErr.ErrorMessage(),
			Fault:   apiErr.ErrorFault().String(),
		}
	}
	return err
}
