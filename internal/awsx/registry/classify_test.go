// SPDX-License-Identifier: MIT

package registry

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		operation                     string
		wantMutating, wantDestructive bool
	}{
		{"GetObject", false, false},
		{"ListBuckets", false, false},
		{"DescribeInstances", false, false},
		// FilterLogEvents (cloudwatchlogs) is the one real "Filter*"
		// operation in the whole catalog, and it's a pure read-only search
		// — this pins down the regression where "Filter" was missing from
		// readPrefixes and the operation defaulted to Mutating=true,
		// incorrectly blocking it under AWS_READONLY=true.
		{"FilterLogEvents", false, false},
		{"PutObject", true, false},
		{"RunInstances", true, false},
		{"DeleteBucket", true, true},
		{"TerminateInstances", true, true},
	}
	for _, c := range cases {
		mutating, destructive := classify(c.operation)
		if mutating != c.wantMutating || destructive != c.wantDestructive {
			t.Errorf("classify(%q) = (mutating=%v, destructive=%v), want (mutating=%v, destructive=%v)",
				c.operation, mutating, destructive, c.wantMutating, c.wantDestructive)
		}
	}
}

// TestClassifyFilterLogEventsInCatalog confirms the fix end-to-end against
// the real cloudwatchlogs.FilterLogEvents operation, not just the classify()
// unit above.
func TestClassifyFilterLogEventsInCatalog(t *testing.T) {
	cat := Build(Factories)
	op, ok := cat.Operation("cloudwatchlogs", "FilterLogEvents")
	if !ok {
		t.Fatal("missing cloudwatchlogs.FilterLogEvents")
	}
	if op.Mutating {
		t.Error("cloudwatchlogs.FilterLogEvents: Mutating=true, want false (it's a read-only log search, would be incorrectly blocked under AWS_READONLY=true)")
	}
}

// FuzzClassify checks classify never panics on arbitrary operation-name
// input and holds its one documented invariant: Destructive is only ever
// true when Mutating is also true (Destructive further narrows Mutating,
// see OperationSpec's doc comment — a destructive-but-not-mutating result
// would be internally inconsistent).
func FuzzClassify(f *testing.F) {
	for _, seed := range []string{
		"", "GetObject", "FilterLogEvents", "PutObject", "DeleteBucket",
		"TerminateInstances", "get", "DELETE", "😀Delete", strings.Repeat("Delete", 100),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, operation string) {
		mutating, destructive := classify(operation)
		if destructive && !mutating {
			t.Fatalf("classify(%q) = (mutating=false, destructive=true), want destructive to imply mutating", operation)
		}
	})
}
