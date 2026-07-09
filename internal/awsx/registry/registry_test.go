// SPDX-License-Identifier: MIT

package registry

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

// emptyClient exposes no methods matching the AWS operation shape at all —
// a stand-in for a hypothetical service package whose client, for whatever
// reason, discovers zero operations.
type emptyClient struct{}

// namedOnlyClient has exactly one method, but it doesn't match the
// operation shape (wrong signature), so it must be discovered as zero
// operations too, not misclassified as one.
type namedOnlyClient struct{}

func (namedOnlyClient) Options() string { return "" }

// TestBuildHandlesZeroOperationService confirms Build doesn't panic and
// still registers the service (with an empty, non-nil Operations map)
// when a factory's client has no operation-shaped methods — an edge case
// no real AWS SDK v2 client hits today, but nothing here should assume
// every service contributes at least one operation.
func TestBuildHandlesZeroOperationService(t *testing.T) {
	factories := map[string]ClientFactory{
		"empty": func(awssdk.Config) any { return emptyClient{} },
		"named": func(awssdk.Config) any { return namedOnlyClient{} },
	}

	cat := Build(factories)

	for _, name := range []string{"empty", "named"} {
		svc, ok := cat.Service(name)
		if !ok {
			t.Fatalf("Service(%q) not found in catalog", name)
		}
		if svc.Operations == nil {
			t.Errorf("%s: Operations is nil, want an empty non-nil map", name)
		}
		if len(svc.Operations) != 0 {
			t.Errorf("%s: Operations = %v, want empty", name, svc.Operations)
		}
		if names := svc.OperationNames(); len(names) != 0 {
			t.Errorf("%s: OperationNames() = %v, want empty", name, names)
		}
	}

	if names := cat.ServiceNames(); len(names) != 2 {
		t.Errorf("ServiceNames() = %v, want 2 entries", names)
	}

	if _, ok := cat.Operation("empty", "AnyOperation"); ok {
		t.Error("Operation lookup on a zero-operation service should report not-found, not panic")
	}
}
