// SPDX-License-Identifier: MIT

package registry

import (
	"reflect"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

// TestDiscoveredOperationsMatchDispatchContract sweeps every operation in
// the real, full catalog (all 426 services) and checks that the client
// method backing it has exactly the shape package dispatch assumes when it
// does:
//
//	method := reflect.ValueOf(client).MethodByName(operation)
//	results := method.Call([]reflect.Value{reflect.ValueOf(ctx), inPtr})
//
// discoverOperations/methodOperationSpec already filter to this shape before
// an operation ever enters the catalog, so this should hold by
// construction. This test exists to catch a divergence between the two
// packages' assumptions directly — e.g. if a future edit to
// methodOperationSpec's checks (reflect.go) loosened what's accepted in a
// way that no longer matches what dispatch.go actually calls, nothing else
// in the suite would notice since dispatch_test.go only exercises a fake
// client, never the real generated client set.
func TestDiscoveredOperationsMatchDispatchContract(t *testing.T) {
	cat := Build(Factories)

	zeroCfg := awssdk.Config{}
	checked := 0
	for _, svcName := range cat.ServiceNames() {
		svc, _ := cat.Service(svcName)
		client := Factories[svcName](zeroCfg)
		clientVal := reflect.ValueOf(client)

		for _, opName := range svc.OperationNames() {
			op := svc.Operations[opName]
			method := clientVal.MethodByName(opName)
			if !method.IsValid() {
				t.Errorf("%s.%s: cataloged but no such method on the client", svcName, opName)
				continue
			}

			mt := method.Type() // bound method value: receiver already applied
			// NumIn()==3: ctx, input, and the variadic optFns slice (counted
			// as one In() even though callers pass zero or more of them).
			if mt.NumIn() != 3 || !mt.IsVariadic() {
				t.Errorf("%s.%s: method shape is (%d args, variadic=%v), dispatch.Invoke calls it with exactly (ctx, input)",
					svcName, opName, mt.NumIn(), mt.IsVariadic())
				continue
			}
			if mt.In(0) != ctxType {
				t.Errorf("%s.%s: first arg is %s, not context.Context", svcName, opName, mt.In(0))
			}
			wantIn := reflect.PointerTo(op.InputType)
			if mt.In(1) != wantIn {
				t.Errorf("%s.%s: second arg is %s, cataloged InputType pointer is %s", svcName, opName, mt.In(1), wantIn)
			}
			if mt.NumOut() != 2 || mt.Out(1) != errType {
				t.Errorf("%s.%s: return shape is (%d values), dispatch.Invoke expects (output, error)", svcName, opName, mt.NumOut())
				continue
			}
			wantOut := reflect.PointerTo(op.OutputType)
			if mt.Out(0) != wantOut {
				t.Errorf("%s.%s: first return is %s, cataloged OutputType pointer is %s", svcName, opName, mt.Out(0), wantOut)
			}
			checked++
		}
	}
	t.Logf("checked=%d operations against the real generated client set", checked)
}
