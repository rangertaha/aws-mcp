// SPDX-License-Identifier: MIT

// Package dispatch invokes any cataloged AWS SDK v2 operation generically, by
// reflection, from a service name, operation name, and raw JSON input. It is
// the runtime counterpart to package registry's compile-time-free operation
// discovery: registry answers "what operations exist and what do they look
// like", dispatch answers "call one of them."
package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/rangertaha/aws-mcp/internal/awsx"
	"github.com/rangertaha/aws-mcp/internal/awsx/registry"
)

// Invoke calls a single AWS operation by service/operation name. It uses
// mgr's active profile to obtain the SDK client and cat's reflected
// operation metadata to decode input and encode output generically — no
// per-operation code is involved.
//
// When readOnly is true, any operation classified as registry.OperationSpec.
// Mutating is refused before an AWS client is even built. Callers reading
// via MCP Resources should always pass readOnly=true regardless of the
// server's configured mode, since resources must never mutate state; tool
// calls pass the server's configured read-only setting.
func Invoke(ctx context.Context, mgr *awsx.Manager, cat *registry.Catalog, service, operation string, input json.RawMessage, readOnly bool) (json.RawMessage, error) {
	op, ok := cat.Operation(service, operation)
	if !ok {
		return nil, fmt.Errorf("unknown operation %s.%s", service, operation)
	}
	if op.Unsupported {
		return nil, fmt.Errorf("%s.%s is not supported by generic dispatch: %s", service, operation, op.UnsupportedReason)
	}
	if readOnly && op.Mutating {
		return nil, fmt.Errorf("%s.%s is a mutating operation and the server is running read-only", service, operation)
	}

	client, err := mgr.Client(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("building client for %s: %w", service, err)
	}

	inPtr := reflect.New(op.InputType)
	if len(input) > 0 {
		if err := json.Unmarshal(input, inPtr.Interface()); err != nil {
			return nil, fmt.Errorf("decoding input for %s.%s: %w", service, operation, err)
		}
	}

	method := reflect.ValueOf(client).MethodByName(operation)
	if !method.IsValid() {
		return nil, fmt.Errorf("operation %s.%s not found on client", service, operation)
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(ctx), inPtr})
	if errVal, _ := results[1].Interface().(error); errVal != nil {
		return nil, mapError(errVal)
	}

	out, err := json.Marshal(results[0].Interface())
	if err != nil {
		return nil, fmt.Errorf("encoding output for %s.%s: %w", service, operation, err)
	}
	return out, nil
}
