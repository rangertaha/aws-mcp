// SPDX-License-Identifier: MIT

package dispatch

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
)

func TestMapErrorExtractsAPIError(t *testing.T) {
	orig := &smithy.GenericAPIError{Code: "NoSuchBucket", Message: "bucket missing", Fault: smithy.FaultClient}

	got := mapError(orig)

	var apiErr *APIError
	if !errors.As(got, &apiErr) {
		t.Fatalf("mapError(%v) = %T, want *APIError", orig, got)
	}
	if apiErr.Code != "NoSuchBucket" || apiErr.Message != "bucket missing" || apiErr.Fault != "client" {
		t.Fatalf("unexpected APIError fields: %+v", apiErr)
	}
	if want := "NoSuchBucket: bucket missing"; apiErr.Error() != want {
		t.Fatalf("Error() = %q, want %q", apiErr.Error(), want)
	}
}

func TestMapErrorPassesThroughNonAPIError(t *testing.T) {
	orig := errors.New("network timeout")

	if got := mapError(orig); !errors.Is(got, orig) {
		t.Fatalf("mapError(%v) = %v, want the original error unchanged", orig, got)
	}
}
