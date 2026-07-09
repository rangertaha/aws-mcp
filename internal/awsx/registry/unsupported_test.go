// SPDX-License-Identifier: MIT

package registry

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestUnsupportedNestedCases exercises specific, real operations chosen to
// pin down the input/output asymmetry in unsupported(): a union type nested
// inside another struct on the input side must still be caught (dispatch
// would otherwise accept the call and fail with a raw json.Unmarshal error
// instead of a clean rejection), while the same shape on the output side
// must NOT be flagged, since json.Marshal follows an interface field's
// concrete runtime value regardless of its static type.
func TestUnsupportedNestedCases(t *testing.T) {
	cat := Build(Factories)

	checks := map[string]bool{
		// TransactItems[].Put.Item, .Delete.Key, etc. are
		// map[string]types.AttributeValue nested inside TransactWriteItem —
		// unsupported only if input recursion reaches through the slice of
		// structs.
		"TransactWriteItems": true,
		"BatchWriteItem":     true,
		"TransactGetItems":   true,
		"ExecuteTransaction": true,
		// Already caught even by a shallow, non-recursive check (top-level
		// union-typed field), kept here as a no-regression anchor.
		"BatchGetItem":     true,
		"ExecuteStatement": true,
		"PutItem":          true,
		"GetItem":          true,
		// A plain string continuation token, no AttributeValue anywhere:
		// must stay dispatchable.
		"ListTables": false,
	}
	for op, wantUnsupported := range checks {
		spec, ok := cat.Operation("dynamodb", op)
		if !ok {
			t.Fatalf("missing dynamodb.%s", op)
		}
		if spec.Unsupported != wantUnsupported {
			t.Errorf("dynamodb.%s: Unsupported=%v reason=%q, want %v", op, spec.Unsupported, spec.UnsupportedReason, wantUnsupported)
		}
	}

	// AnalyzerSummary.Configuration (a union type) is nested inside
	// GetAnalyzerOutput.Analyzer, but only on the OUTPUT side: it must stay
	// dispatchable, since a populated union value marshals fine.
	spec, ok := cat.Operation("accessanalyzer", "GetAnalyzer")
	if !ok {
		t.Fatal("missing accessanalyzer.GetAnalyzer")
	}
	if spec.Unsupported {
		t.Errorf("accessanalyzer.GetAnalyzer: Unsupported=true, want false (output-only nested union marshals fine): reason=%q", spec.UnsupportedReason)
	}

	// bedrockruntime.InvokeModelWithBidirectionalStream stores its
	// genuinely undispatchable event-stream channel in UNEXPORTED output
	// fields (eventStream, initialReply) — these must still be checked, not
	// skipped as "internal implementation detail".
	spec, ok = cat.Operation("bedrockruntime", "InvokeModelWithBidirectionalStream")
	if !ok {
		t.Fatal("missing bedrockruntime.InvokeModelWithBidirectionalStream")
	}
	if !spec.Unsupported {
		t.Error("bedrockruntime.InvokeModelWithBidirectionalStream: Unsupported=false, want true (unexported channel field)")
	}
}

// oldShallowReason reproduces the pre-fix detector's per-field
// classification (every field regardless of export, no struct recursion),
// tagging *why* it considered a field unsupported so
// TestUnsupportedFixHasNoGenuineRegressions can tell a deliberate
// input/output-asymmetry relaxation from an actual regression.
func oldShallowReason(ft reflect.Type) string {
	switch ft.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return oldShallowReason(ft.Elem())
	case reflect.Chan:
		return "chan"
	case reflect.Interface:
		if ft.NumMethod() == 0 {
			return ""
		}
		jm := reflect.TypeFor[json.Marshaler]()
		ju := reflect.TypeFor[json.Unmarshaler]()
		if ft.Implements(jm) && ft.Implements(ju) {
			return ""
		}
		if ft.Implements(readerType) || ft.Implements(writerType) {
			return "stream"
		}
		return "plain-interface"
	}
	if ft.Implements(readerType) || ft.Implements(writerType) {
		return "stream"
	}
	return ""
}

func oldShallowStructReason(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		if r := oldShallowReason(t.Field(i).Type); r != "" {
			return r
		}
	}
	return ""
}

// TestUnsupportedFixHasNoGenuineRegressions sweeps the entire catalog
// comparing the current (recursive, input/output-asymmetric) unsupported()
// against a faithful reproduction of the original shallow, symmetric
// detector. The only operations allowed to flip from unsupported to
// supported are ones whose *only* old reason was an output-side
// plain-interface (union) field — every other kind of flip (a stream, a
// channel, or any input-side reason no longer being caught) is a genuine
// regression and fails the test.
func TestUnsupportedFixHasNoGenuineRegressions(t *testing.T) {
	cat := Build(Factories)

	total, oldUnsupported, newUnsupported, newlyCaught, intentionallyRelaxed := 0, 0, 0, 0, 0
	for _, svcName := range cat.ServiceNames() {
		svc, _ := cat.Service(svcName)
		for _, op := range svc.Operations {
			total++
			oldInReason := oldShallowStructReason(op.InputType)
			oldOutReason := oldShallowStructReason(op.OutputType)
			oldBad := oldInReason != "" || oldOutReason != ""
			if oldBad {
				oldUnsupported++
			}
			if op.Unsupported {
				newUnsupported++
			}

			switch {
			case !oldBad && op.Unsupported:
				newlyCaught++
			case oldBad && !op.Unsupported:
				if oldInReason != "" {
					t.Errorf("regression: %s.%s old input reason=%q now marked supported", svcName, op.Name, oldInReason)
				} else if oldOutReason != "plain-interface" {
					t.Errorf("regression: %s.%s old output reason=%q now marked supported", svcName, op.Name, oldOutReason)
				} else {
					intentionallyRelaxed++
				}
			}
		}
	}
	t.Logf("total=%d old_unsupported=%d new_unsupported=%d newly_caught=%d intentionally_relaxed=%d",
		total, oldUnsupported, newUnsupported, newlyCaught, intentionallyRelaxed)
}
