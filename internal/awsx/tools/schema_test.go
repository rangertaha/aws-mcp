// SPDX-License-Identifier: MIT

package tools

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

// hasCyclicType reports whether t's type graph (through pointers, slices,
// arrays, and maps) reaches a named struct type already on the current
// recursion stack — a genuine, unbounded, self-referential type (e.g.
// wafv2's Statement, nested via And/Or/Not sub-statements).
func hasCyclicType(t reflect.Type, path map[reflect.Type]bool) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return hasCyclicType(t.Elem(), path)
	case reflect.Struct:
		if path[t] {
			return true
		}
		path[t] = true
		defer delete(path, t)
		for i := 0; i < t.NumField(); i++ {
			if hasCyclicType(t.Field(i).Type, path) {
				return true
			}
		}
	}
	return false
}

// TestOperationSchemaHandlesCyclicTypes finds every genuinely
// self-referential Input/Output type across the whole catalog and confirms
// operationSchema (what aws_describe_operation actually calls) neither
// errors, hangs, nor panics on any of them — jsonschema.ForType has no way
// to express an unbounded type as a finite JSON Schema and errors out on
// its own, so cyclicTypeOverrides must break every one of these before
// calling it.
func TestOperationSchemaHandlesCyclicTypes(t *testing.T) {
	cat := registry.Build(registry.Factories)

	type target struct {
		service, op, side string
		typ               reflect.Type
	}
	var targets []target
	for _, svcName := range cat.ServiceNames() {
		svc, _ := cat.Service(svcName)
		for _, op := range svc.Operations {
			if hasCyclicType(op.InputType, map[reflect.Type]bool{}) {
				targets = append(targets, target{svcName, op.Name, "input", op.InputType})
			}
			if hasCyclicType(op.OutputType, map[reflect.Type]bool{}) {
				targets = append(targets, target{svcName, op.Name, "output", op.OutputType})
			}
		}
	}
	if len(targets) == 0 {
		t.Fatal("no cyclic types found in the catalog — either the SDK changed or hasCyclicType is broken; this test needs real cyclic types to be meaningful")
	}
	t.Logf("testing %d cyclic-type targets", len(targets))

	sawOverridePlaceholder := false
	for _, tg := range targets {
		done := make(chan struct{})
		var schema []byte
		var schemaErr error
		var panicked any
		go func() {
			defer close(done)
			defer func() { panicked = recover() }()
			schema, schemaErr = operationSchema(tg.typ)
		}()
		select {
		case <-done:
			if panicked != nil {
				t.Errorf("%s.%s (%s): operationSchema PANICKED: %v", tg.service, tg.op, tg.side, panicked)
				continue
			}
			if schemaErr != nil {
				t.Errorf("%s.%s (%s): operationSchema error: %v", tg.service, tg.op, tg.side, schemaErr)
				continue
			}
			if len(schema) == 0 {
				t.Errorf("%s.%s (%s): operationSchema returned no error but an empty schema", tg.service, tg.op, tg.side)
			}
			// Confirm success comes from cyclicTypeOverrides actually
			// breaking the cycle, not some incidental path (e.g.
			// IgnoreInvalidTypes alone does NOT suppress jsonschema-go's
			// cycle error — verified separately; this placeholder text is
			// only present if our override map was consulted).
			if strings.Contains(string(schema), "recursive type") {
				sawOverridePlaceholder = true
			}
		case <-time.After(3 * time.Second):
			t.Errorf("%s.%s (%s): operationSchema HUNG on %v", tg.service, tg.op, tg.side, tg.typ)
		}
	}
	if !sawOverridePlaceholder {
		t.Error("no generated schema contained the cyclicTypeOverrides placeholder text — the override mechanism may not actually be engaging")
	}
}
