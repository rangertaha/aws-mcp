// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"reflect"
)

var (
	ctxType = reflect.TypeFor[context.Context]()
	errType = reflect.TypeFor[error]()
)

// discoverOperations enumerates the exported methods on client matching the
// AWS SDK v2 operation shape:
//
//	func(context.Context, *XInput, ...func(*Options)) (*XOutput, error)
//
// This excludes every other exported method AWS SDK v2 clients expose (e.g.
// Options()) without any per-service special-casing: reflect.Type.NumMethod
// only enumerates exported methods regardless of caller package, and the
// signature shape checked here is specific enough that no known non-operation
// method matches it.
func discoverOperations(service string, client any) []*OperationSpec {
	t := reflect.TypeOf(client)
	var ops []*OperationSpec
	for i := 0; i < t.NumMethod(); i++ {
		if spec, ok := methodOperationSpec(service, t.Method(i)); ok {
			ops = append(ops, spec)
		}
	}
	return ops
}

// methodOperationSpec builds an OperationSpec from a client method if, and
// only if, it matches the standard operation shape.
func methodOperationSpec(service string, m reflect.Method) (*OperationSpec, bool) {
	ft := m.Func.Type() // method expression: receiver is In(0)

	// (receiver, ctx, *Input, ...func(*Options)) (*Output, error)
	if ft.NumIn() != 4 || !ft.IsVariadic() {
		return nil, false
	}
	if ft.In(1) != ctxType {
		return nil, false
	}

	inputType := ft.In(2)
	if inputType.Kind() != reflect.Pointer || inputType.Elem().Kind() != reflect.Struct {
		return nil, false
	}

	if ft.NumOut() != 2 || ft.Out(1) != errType {
		return nil, false
	}
	outputType := ft.Out(0)
	if outputType.Kind() != reflect.Pointer || outputType.Elem().Kind() != reflect.Struct {
		return nil, false
	}

	spec := &OperationSpec{
		Service:    service,
		Name:       m.Name,
		InputType:  inputType.Elem(),
		OutputType: outputType.Elem(),
	}
	spec.Mutating, spec.Destructive = classify(m.Name)
	spec.Unsupported, spec.UnsupportedReason = unsupported(spec.InputType, spec.OutputType)
	spec.PaginationField = paginationField(spec.OutputType)
	return spec, true
}
